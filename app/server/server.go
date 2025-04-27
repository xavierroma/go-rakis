package server

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/http-server-starter-go/app/types"
)

// RequestHandler is a function that processes HTTP requests
type RequestHandler func(ctx context.Context, req types.Request) types.Response

type Server struct {
	addr    string
	handler RequestHandler
}

type Error error

func NewServer(addr string) *Server {
	return &Server{
		addr: addr,
	}
}

func (s *Server) WithHandler(h RequestHandler) *Server {
	s.handler = h
	return s
}

func (s Server) Listen() (net.Listener, Error) {
	l, err := net.Listen("tcp", s.addr)
	if err != nil {
		fmt.Println("Failed to start")
		return nil, err
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	req, err := parseRequest(conn)
	if err != nil {
		fmt.Println("Failed to parse request:", err)
		errorRes := prepareResponse(types.Request{})
		errorRes.Status = types.StatusBadRequest
		errorRes.Headers["Connection"] = "close"
		respond(conn, types.Request{Headers: map[string]string{"Connection": "close"}}, errorRes)
		return
	}

	res := s.handler(context.Background(), req)
	respond(conn, req, res)
}

func parseRequest(conn net.Conn) (types.Request, Error) {
	result := types.Request{
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
	result.Method = types.Method(string(requestLineParts[0]))
	result.Target = string(requestLineParts[1])
	result.Version = string(requestLineParts[2])

	if result.Version != "HTTP/1.1" {
		return result, fmt.Errorf("unsupported HTTP version: %q", result.Version)
	}

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

func prepareResponse(r types.Request) types.Response {
	return types.Response{
		Status:     types.StatusOK,
		Headers:    map[string]string{"Server": "go-server/0.1", "Date": time.Now().UTC().Format(time.RFC1123)},
		Body:       nil,
		BodyReader: nil,
	}
}

func respond(conn net.Conn, req types.Request, r types.Response) {
	crlf := []byte("\r\n")

	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}

	rspMap := map[types.Status]string{
		types.StatusOK:                  "HTTP/1.1 200 OK",
		types.StatusNotFound:            "HTTP/1.1 404 Not Found",
		types.StatusBadRequest:          "HTTP/1.1 400 Bad Request",
		types.StatusInternalServerError: "HTTP/1.1 500 Internal Server Error",
		types.StatusCreated:             "HTTP/1.1 201 Created",
	}

	connectionHeader := "keep-alive"
	isErrorStatus := r.Status == types.StatusBadRequest || r.Status == types.StatusNotFound || r.Status == types.StatusInternalServerError
	if req.Headers["Connection"] == "close" || isErrorStatus {
		connectionHeader = "close"
	}
	r.Headers["Connection"] = connectionHeader

	isChunked := r.BodyReader != nil
	var bodyToWrite []byte = r.Body

	if !isChunked {
		if _, ok := r.Headers["Content-Length"]; !ok && r.Body != nil {
			r.Headers["Content-Length"] = strconv.Itoa(len(r.Body))
		} else if !ok && r.Body == nil {
			r.Headers["Content-Length"] = "0"
		}

		canUseGzip := false
		if acceptEncoding, ok := req.Headers["Accept-Encoding"]; ok {
			if strings.Contains(acceptEncoding, "gzip") {
				canUseGzip = true
			}
		}

		if canUseGzip && r.Body != nil {
			var buf bytes.Buffer
			gz := gzip.NewWriter(&buf)
			if _, err := gz.Write(r.Body); err == nil {
				if err := gz.Close(); err == nil {
					bodyToWrite = buf.Bytes()
					r.Headers["Content-Encoding"] = "gzip"
					r.Headers["Content-Length"] = strconv.Itoa(len(bodyToWrite))
				} else {
					fmt.Println("Error closing gzip writer:", err)
				}
			} else {
				fmt.Println("Error writing to gzip writer:", err)
			}
		}
	} else {
		r.Headers["Transfer-Encoding"] = "chunked"
		delete(r.Headers, "Content-Length")
	}

	statusLine := rspMap[r.Status]
	if _, err := conn.Write([]byte(statusLine)); err != nil {
		fmt.Println("Error writing status line:", err)
		return
	}
	if _, err := conn.Write(crlf); err != nil {
		fmt.Println("Error writing CRLF after status line:", err)
		return
	}

	for k, v := range r.Headers {
		headerLine := fmt.Sprintf("%s: %s", k, v)
		if _, err := conn.Write([]byte(headerLine)); err != nil {
			fmt.Println("Error writing header:", k, v, err)
			return
		}
		if _, err := conn.Write(crlf); err != nil {
			fmt.Println("Error writing CRLF after header:", k, v, err)
			return
		}
	}

	if _, err := conn.Write(crlf); err != nil {
		fmt.Println("Error writing CRLF after headers:", err)
		return
	}

	if isChunked {
		buf := make([]byte, 4*1024)
		for {
			n, err := r.BodyReader.Read(buf)
			if n > 0 {
				chunkSizeHex := []byte(strconv.FormatInt(int64(n), 16))
				if _, wErr := conn.Write(chunkSizeHex); wErr != nil {
					fmt.Println("Error writing chunk size:", wErr)
					return
				}
				if _, wErr := conn.Write(crlf); wErr != nil {
					fmt.Println("Error writing CRLF after chunk size:", wErr)
					return
				}

				if _, wErr := conn.Write(buf[:n]); wErr != nil {
					fmt.Println("Error writing chunk data:", wErr)
					return
				}
				if _, wErr := conn.Write(crlf); wErr != nil {
					fmt.Println("Error writing CRLF after chunk data:", wErr)
					return
				}
			}

			if err != nil {
				if err == io.EOF {
					if _, wErr := conn.Write([]byte("0")); wErr != nil {
						fmt.Println("Error writing zero chunk size:", wErr)
						return
					}
					if _, wErr := conn.Write(crlf); wErr != nil {
						fmt.Println("Error writing CRLF after zero chunk size:", wErr)
						return
					}
					if _, wErr := conn.Write(crlf); wErr != nil {
						fmt.Println("Error writing final CRLF for chunked:", wErr)
						return
					}
				} else {
					fmt.Println("Error reading from body reader:", err)
				}
				break
			}
		}
	} else if bodyToWrite != nil {
		if _, err := conn.Write(bodyToWrite); err != nil {
			fmt.Println("Error writing non-chunked body:", err)
			return
		}
	}
}
