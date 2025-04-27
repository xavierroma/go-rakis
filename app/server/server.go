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
)

type Server struct {
	addr     string
	handlers map[Method]map[string]Handler
}

type Method string

const (
	Get    Method = "GET"
	Post   Method = "POST"
	Put    Method = "PUT"
	Patch  Method = "PATCH"
	Delete Method = "DELETE"
)

type Request struct {
	Method  Method
	Version string
	Target  string
	Headers map[string]string
	Body    *string
	Params  map[string]string
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

type Handler func(ctx context.Context, req Request, res *Response)

type Error error

func NewServer(addr string) Server {
	return Server{
		addr:     addr,
		handlers: make(map[Method]map[string]Handler),
	}
}

func (s Server) RegisterHandler(m Method, p string, h Handler) Server {
	_, ok := s.handlers[m]
	if !ok {
		s.handlers[m] = make(map[string]Handler, 1)
	}
	s.handlers[m][p] = h
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
		errorRes := prepareResponse(Request{})
		errorRes.Status = StatusBadRequest
		errorRes.Headers["Connection"] = "close"
		respond(conn, Request{Headers: map[string]string{"Connection": "close"}}, errorRes)
		return
	}

	handle, ok := findHandler(s.handlers[Method(req.Method)], &req)
	if !ok {
		res := prepareResponse(req)
		res.Status = StatusNotFound
		respond(conn, req, res)
		return
	}

	res := prepareResponse(req)

	(*handle)(context.Background(), req, &res)

	respond(conn, req, res)
}

func parseRequest(conn net.Conn) (Request, Error) {
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
	result.Method = Method(string(requestLineParts[0]))
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

func prepareResponse(r Request) Response {
	return Response{
		Status: StatusOK,
		Headers: map[string]string{
			"Server": "go-server/0.1",
			"Date":   time.Now().UTC().Format(time.RFC1123),
		},
		Body: nil,
	}
}

func respond(conn net.Conn, req Request, r Response) {
	crlf := []byte("\r\n")

	rspMap := map[Status]string{
		StatusOK:                  "HTTP/1.1 200 OK",
		StatusNotFound:            "HTTP/1.1 404 Not Found",
		StatusBadRequest:          "HTTP/1.1 400 Bad Request",
		StatusInternalServerError: "HTTP/1.1 500 Internal Server Error",
		StatusCreated:             "HTTP/1.1 201 Created",
	}

	if _, ok := r.Headers["Content-Length"]; !ok && r.Body != nil {
		r.Headers["Content-Length"] = strconv.Itoa(len(r.Body))
	}

	connectionHeader := "keep-alive"
	if req.Headers["Connection"] == "close" || r.Status >= 400 {
		connectionHeader = "close"
	}
	r.Headers["Connection"] = connectionHeader

	canUseGzip := false
	if acceptEncoding, ok := req.Headers["Accept-Encoding"]; ok {
		if strings.Contains(acceptEncoding, "gzip") {
			canUseGzip = true
		}
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

	var bodyToWrite []byte = r.Body
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

	if bodyToWrite != nil {
		if _, err := conn.Write(bodyToWrite); err != nil {
			fmt.Println("Error writing body:", err)
			return
		}
	}
}

func findHandler(handlers map[string]Handler, req *Request) (*Handler, bool) {
	targetSegments := strings.Split(req.Target, "/")
Outer:
	for path, handler := range handlers {
		handlerSegments := strings.Split(path, "/")
		if len(targetSegments) != len(handlerSegments) {
			continue
		}
		params := make(map[string]string, 0)
		for i := 0; i < len(handlerSegments); i++ {
			isParam := strings.HasPrefix(handlerSegments[i], ":")
			if isParam {
				params[strings.Replace(handlerSegments[i], ":", "", 1)] = targetSegments[i]
				continue
			}
			if handlerSegments[i] != targetSegments[i] {
				continue Outer
			}
		}
		req.Params = params
		return &handler, true
	}

	return nil, false
}
