package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

var directory string

func main() {
	flag.StringVar(&directory, "directory", "/tmp", "directory to serve files from")
	flag.Parse()

	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		fmt.Println("Waiting for connection...")
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		fmt.Println("Connection accepted")
		go handler(context.Background(), conn)
	}
}

func handler(ctx context.Context, conn net.Conn) {
	fmt.Println("Handling request...")
	for {
		select {
		case <-ctx.Done():
			conn.Close()
			return
		default:
			req, err := handleRequest(ctx, conn)
			if err != nil {
				if errors.Is(err, io.EOF) {
					fmt.Println("Client closed connection.")
				} else {
					fmt.Println("Error handling request:", err)
				}
				conn.Close()
				return
			}
			fmt.Println("Request handled:", req)
			fmt.Println("Handling response...")
			res, err := handleResponse(ctx, conn, req)
			if err != nil {
				fmt.Println("Error handling response:", err)
				conn.Close()
				return
			}
			respond(ctx, conn, req, res)
			fmt.Println("Response handled")
			if req.Headers["Connection"] == "close" {
				conn.Close()
				return
			}
		}
	}
}

type Request struct {
	Method  string
	Version string
	Target  string
	Headers map[string]string
	Body    *string
}

func handleRequest(ctx context.Context, conn net.Conn) (Request, error) {
	result := Request{
		Headers: make(map[string]string),
		Body:    nil,
	}
	reader := bufio.NewReader(conn)

	requestLineBytes, err := reader.ReadBytes('\n')
	if err != nil {
		return result, fmt.Errorf("error reading request line: %w", err)
	}
	requestLineBytes = bytes.TrimRight(requestLineBytes, "\r\n")
	if len(requestLineBytes) == 0 {
		return result, errors.New("empty request line")
	}

	requestLineParts := bytes.SplitN(requestLineBytes, []byte(" "), 3)
	if len(requestLineParts) != 3 {
		return result, fmt.Errorf("malformed request line: %q", string(requestLineBytes))
	}
	result.Method = string(requestLineParts[0])
	result.Target = string(requestLineParts[1])
	result.Version = string(requestLineParts[2])

	for {
		headerLineBytes, err := reader.ReadBytes('\n')
		if err != nil {
			return result, fmt.Errorf("error reading header line: %w", err)
		}

		headerLineBytes = bytes.TrimRight(headerLineBytes, "\r\n")

		if len(headerLineBytes) == 0 {
			break
		}

		headerParts := bytes.SplitN(headerLineBytes, []byte(":"), 2)
		if len(headerParts) != 2 {
			fmt.Printf("Warning: Skipping malformed header line: %q\n", string(headerLineBytes))
			continue
		}

		key := strings.TrimSpace(string(headerParts[0]))
		value := strings.TrimSpace(string(headerParts[1]))

		result.Headers[key] = value
	}

	if result.Method == "POST" {
		if contentLengthStr, ok := result.Headers["Content-Length"]; ok {
			contentLength, err := strconv.Atoi(contentLengthStr)
			if err != nil {
				return result, fmt.Errorf("invalid Content-Length: %w", err)
			}
			bodyBytes := make([]byte, contentLength)
			_, err = io.ReadFull(reader, bodyBytes)
			if err != nil {
				return result, fmt.Errorf("error reading request body: %w", err)
			}
			bodyStr := string(bodyBytes)
			result.Body = &bodyStr
		}
	}

	return result, nil
}

type Status int

const (
	StatusOK Status = iota
	StatusNotFound
	StatusBadRequest
	StatusInternalServerError
	StatusCreated
)

type Response struct {
	Status  Status
	Body    []byte
	Headers map[string]string
}

func handleResponse(ctx context.Context, conn net.Conn, req Request) (Response, error) {
	res := Response{
		Headers: make(map[string]string),
	}
	if req.Method == "GET" && req.Target == "/" {
		res.Status = StatusOK
		res.Body = []byte("Hello, World!")
		res.Headers["Content-Type"] = "text/plain"
		res.Headers["Content-Length"] = fmt.Sprintf("%d", len(res.Body))
	} else if req.Method == "GET" && strings.HasPrefix(req.Target, "/echo") {
		res.Status = StatusOK
		echoTarget := strings.TrimPrefix(req.Target, "/echo/")
		res.Body = []byte(echoTarget)
		res.Headers["Content-Type"] = "text/plain"
		res.Headers["Content-Length"] = fmt.Sprintf("%d", len(res.Body))
	} else if req.Method == "GET" && req.Target == "/user-agent" {
		res.Status = StatusOK
		res.Body = []byte(req.Headers["User-Agent"])
		res.Headers["Content-Type"] = "text/plain"
		res.Headers["Content-Length"] = fmt.Sprintf("%d", len(res.Body))
	} else if req.Method == "GET" && strings.HasPrefix(req.Target, "/files/") {
		res.Status = StatusOK
		path := directory + "/" + strings.TrimPrefix(req.Target, "/files/")
		_, err := os.Stat(path)
		if err != nil {
			res.Status = StatusNotFound
			res.Headers["Content-Length"] = "0"
		} else {
			body, err := os.ReadFile(path)
			if err != nil {
				fmt.Println("Error reading file:", err)
				return res, fmt.Errorf("error reading file: %w", err)
			}
			res.Headers["Content-Type"] = "application/octet-stream"
			res.Headers["Content-Length"] = fmt.Sprintf("%d", len(body))
		}
	} else if req.Method == "POST" && strings.HasPrefix(req.Target, "/files/") {
		res.Status = StatusCreated
		path := directory + "/" + strings.TrimPrefix(req.Target, "/files/")
		contentLengthStr := req.Headers["Content-Length"]
		_, err := strconv.Atoi(contentLengthStr)
		if err != nil {
			fmt.Println("Error converting Content-Length to integer:", err)
			return res, fmt.Errorf("error converting Content-Length to integer: %w", err)
		}
		if req.Body == nil {
			fmt.Println("Error: POST request has no body")
			res.Status = StatusBadRequest
			res.Headers["Content-Length"] = "0"
		} else {
			err = os.WriteFile(path, []byte(*req.Body), 0o644)
			if err != nil {
				fmt.Println("Error writing file:", err)
				res.Status = StatusInternalServerError
				res.Headers["Content-Length"] = "0"
			} else {
				res.Status = StatusCreated
				res.Headers["Content-Length"] = "0"
			}
		}
	} else {
		res.Status = StatusNotFound
		res.Headers["Content-Length"] = "0"
	}

	return res, nil
}

func respond(ctx context.Context, conn net.Conn, req Request, res Response) {
	crlf := []byte("\r\n")

	rspMap := map[Status]string{
		StatusOK:                  "HTTP/1.1 200 OK",
		StatusNotFound:            "HTTP/1.1 404 Not Found",
		StatusBadRequest:          "HTTP/1.1 400 Bad Request",
		StatusInternalServerError: "HTTP/1.1 500 Internal Server Error",
		StatusCreated:             "HTTP/1.1 201 Created",
	}

	if req.Headers["Connection"] == "close" {
		res.Headers["Connection"] = "close"
	}

	compression := req.Headers["Accept-Encoding"]
	if compression == "gzip" {
		res.Headers["Content-Encoding"] = "gzip"
	}
	rspLine := rspMap[res.Status]
	_, err := conn.Write([]byte(rspLine))
	if err != nil {
		fmt.Println("Error writing status line:", err)
		return
	}
	for k, v := range res.Headers {
		_, err = conn.Write([]byte(fmt.Sprintf("%s: %s", k, v)))
		if err != nil {
			fmt.Println("Error writing header:", k, v, err)
			return
		}
	}
	_, err = conn.Write(crlf)
	if err != nil {
		fmt.Println("Error writing CRLF after headers:", err)
		return
	}

	if res.Body != nil {
		if compression == "gzip" {
			gz, err := gzip.NewWriterLevel(conn, gzip.BestCompression)
			if err != nil {
				fmt.Println("Error creating gzip writer:", err)
				return
			}
			_, err = gz.Write(res.Body)
			if err != nil {
				fmt.Println("Error writing body:", err)
				return
			}
			err = gz.Close()
			if err != nil {
				fmt.Println("Error closing gzip writer:", err)
				return
			}
		} else {
			_, err = conn.Write(res.Body)
			if err != nil {
				fmt.Println("Error writing body:", err)
				return
			}
		}
	}
	_, err = conn.Write(crlf)
	if err != nil {
		fmt.Println("Error writing CRLF after body:", err)
		return
	}
}
