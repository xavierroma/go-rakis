package main

import (
	"bufio"
	"bytes"
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
			handleResponse(ctx, conn, req)
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

func handleResponse(ctx context.Context, conn net.Conn, req Request) {
	crlf := []byte("\r\n")
	var response string
	var body []byte
	var headers []string

	if req.Method == "GET" && req.Target == "/" {
		response = "HTTP/1.1 200 OK"
		body = []byte("Hello, World!")
		headers = append(headers, "Content-Type: text/plain")
		headers = append(headers, fmt.Sprintf("Content-Length: %d", len(body)))
	} else if req.Method == "GET" && strings.HasPrefix(req.Target, "/echo") {
		response = "HTTP/1.1 200 OK"
		echoTarget := strings.TrimPrefix(req.Target, "/echo/")
		body = []byte(echoTarget)
		headers = append(headers, "Content-Type: text/plain")
		headers = append(headers, fmt.Sprintf("Content-Length: %d", len(body)))
	} else if req.Method == "GET" && req.Target == "/user-agent" {
		response = "HTTP/1.1 200 OK"
		body = []byte(req.Headers["User-Agent"])
		headers = append(headers, "Content-Type: text/plain")
		headers = append(headers, fmt.Sprintf("Content-Length: %d", len(body)))
	} else if req.Method == "GET" && strings.HasPrefix(req.Target, "/files/") {
		response = "HTTP/1.1 200 OK"
		path := directory + "/" + strings.TrimPrefix(req.Target, "/files/")
		_, err := os.Stat(path)
		if err != nil {
			response = "HTTP/1.1 404 Not Found"
			headers = append(headers, "Content-Length: 0")
		} else {
			body, err = os.ReadFile(path)
			if err != nil {
				fmt.Println("Error reading file:", err)
				return
			}
			headers = append(headers, "Content-Type: application/octet-stream")
			headers = append(headers, fmt.Sprintf("Content-Length: %d", len(body)))
		}
	} else if req.Method == "POST" && strings.HasPrefix(req.Target, "/files/") {
		response = "HTTP/1.1 201 Created"
		path := directory + "/" + strings.TrimPrefix(req.Target, "/files/")
		contentLengthStr := req.Headers["Content-Length"]
		_, err := strconv.Atoi(contentLengthStr)
		if err != nil {
			fmt.Println("Error converting Content-Length to integer:", err)
			return
		}
		if req.Body == nil {
			fmt.Println("Error: POST request has no body")
			response = "HTTP/1.1 400 Bad Request"
			headers = append(headers, "Content-Length: 0")
		} else {
			err = os.WriteFile(path, []byte(*req.Body), 0o644)
			if err != nil {
				fmt.Println("Error writing file:", err)
				response = "HTTP/1.1 500 Internal Server Error"
				headers = append(headers, "Content-Length: 0")
			} else {
				response = "HTTP/1.1 201 Created"
				headers = append(headers, "Content-Length: 0")
			}
		}
	} else {
		response = "HTTP/1.1 404 Not Found"
		headers = append(headers, "Content-Length: 0")
	}

	if req.Headers["Connection"] == "close" {
		headers = append(headers, "Connection: close")
	}

	_, err := conn.Write([]byte(response))
	if err != nil {
		fmt.Println("Error writing status line:", err)
		return
	}
	_, err = conn.Write(crlf)
	if err != nil {
		fmt.Println("Error writing CRLF after status line:", err)
		return
	}

	for _, h := range headers {
		_, err = conn.Write([]byte(h))
		if err != nil {
			fmt.Println("Error writing header:", h, err)
			return
		}
		_, err = conn.Write(crlf)
		if err != nil {
			fmt.Println("Error writing CRLF after header:", h, err)
			return
		}
	}

	_, err = conn.Write(crlf)
	if err != nil {
		fmt.Println("Error writing header/body separator CRLF:", err)
		return
	}

	if len(body) > 0 {
		_, err = conn.Write(body)
		if err != nil {
			fmt.Println("Error writing body:", err)
			return
		}
	}
}
