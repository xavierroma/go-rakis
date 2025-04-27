package server

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/codecrafters-io/http-server-starter-go/app/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readResponse(conn net.Conn) (statusLine string, headers map[string]string, body []byte, err error) {
	reader := bufio.NewReader(conn)
	headers = make(map[string]string)

	statusLineBytes, err := reader.ReadBytes('\n')
	if err != nil {
		if err == io.EOF && len(statusLineBytes) > 0 { // Allow EOF if status line was read
			statusLine = strings.TrimRight(string(statusLineBytes), "\r\n")
			return statusLine, headers, nil, nil // No headers/body if conn closes early
		} else if err == io.EOF {
			return "", nil, nil, fmt.Errorf("connection closed before status line: %w", io.ErrUnexpectedEOF)
		}
		return "", nil, nil, fmt.Errorf("error reading status line: %w", err)
	}
	statusLine = strings.TrimRight(string(statusLineBytes), "\r\n")

	for {
		headerLineBytes, err := reader.ReadBytes('\n')
		if err != nil {
			return statusLine, headers, nil, fmt.Errorf("error reading header line: %w", err)
		}
		headerLine := strings.TrimRight(string(headerLineBytes), "\r\n")
		if len(headerLine) == 0 {
			break // End of headers
		}
		parts := strings.SplitN(headerLine, ":", 2)
		if len(parts) != 2 {
			return statusLine, headers, nil, fmt.Errorf("malformed header: %q", headerLine)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		headers[key] = value
	}

	if headers["Transfer-Encoding"] == "chunked" {
		var bodyBuffer bytes.Buffer
		for {
			chunkSizeBytes, err := reader.ReadBytes('\n')
			if err != nil {
				return statusLine, headers, nil, fmt.Errorf("error reading chunk size: %w", err)
			}
			chunkSizeHex := strings.TrimRight(string(chunkSizeBytes), "\r\n")
			chunkSize, err := strconv.ParseInt(chunkSizeHex, 16, 64)
			if err != nil {
				return statusLine, headers, nil, fmt.Errorf("error parsing chunk size hex %q: %w", chunkSizeHex, err)
			}

			if chunkSize == 0 {
				_, err = reader.ReadBytes('\n') // Read final CRLF
				if err != nil && err != io.EOF {
					return statusLine, headers, bodyBuffer.Bytes(), fmt.Errorf("error reading final CRLF for chunked: %w", err)
				}
				break
			}

			chunkData := make([]byte, chunkSize)
			_, err = io.ReadFull(reader, chunkData)
			if err != nil {
				return statusLine, headers, bodyBuffer.Bytes(), fmt.Errorf("error reading chunk data (expected %d bytes): %w", chunkSize, err)
			}
			bodyBuffer.Write(chunkData)

			_, err = reader.ReadBytes('\n') // Read CRLF after chunk data
			if err != nil {
				return statusLine, headers, bodyBuffer.Bytes(), fmt.Errorf("error reading CRLF after chunk data: %w", err)
			}
		}
		body = bodyBuffer.Bytes()
	} else if contentLengthStr, ok := headers["Content-Length"]; ok {
		contentLength, err := strconv.Atoi(contentLengthStr)
		if err != nil {
			return statusLine, headers, nil, fmt.Errorf("invalid Content-Length %q: %w", contentLengthStr, err)
		}
		if contentLength > 0 {
			bodyBytes := make([]byte, contentLength)
			_, err = io.ReadFull(reader, bodyBytes)
			if err != nil && err != io.ErrUnexpectedEOF { // Allow UnexpectedEOF if client sends less data than Content-Length
				return statusLine, headers, bodyBytes[:len(bodyBytes)-contentLength+reader.Buffered()], fmt.Errorf("error reading body (expected %d bytes): %w", contentLength, err)
			} else if err == io.ErrUnexpectedEOF {
				body = bodyBytes[:len(bodyBytes)-contentLength+reader.Buffered()]
				return statusLine, headers, body, err // Return UnexpectedEOF if applicable
			}
			body = bodyBytes
		}
	}

	return statusLine, headers, body, nil
}

func mockHandler(response types.Response) RequestHandler {
	return func(ctx context.Context, req types.Request) types.Response {
		return response
	}
}

func TestNewServer(t *testing.T) {
	addr := "localhost:12345"
	s := NewServer(addr)
	require.NotNil(t, s)
	assert.Equal(t, addr, s.addr)
	assert.Nil(t, s.handler)
}

func TestWithHandler(t *testing.T) {
	s := NewServer("localhost:8080")
	h := mockHandler(types.Response{Status: types.StatusOK})
	s.WithHandler(h)
	require.NotNil(t, s.handler)
}

func runHandleConnectionTest(t *testing.T, handler RequestHandler, request string) (string, map[string]string, []byte, error) {
	t.Helper()
	serverConn, clientConn := net.Pipe()
	s := &Server{handler: handler} // Create a minimal server with just the handler

	go func() {
		defer serverConn.Close()
		s.handleConnection(serverConn)
	}()

	// Write request to the client side of the pipe
	_, err := clientConn.Write([]byte(request))
	require.NoError(t, err)

	// Read response from the client side
	status, headers, body, readErr := readResponse(clientConn)
	clientConn.Close() // Ensure client side is closed
	return status, headers, body, readErr
}

func TestHandleConnection_ValidGET(t *testing.T) {
	expectedBody := "Hello GET"
	h := mockHandler(types.Response{
		Status:  types.StatusOK,
		Headers: map[string]string{"Content-Type": "text/plain"},
		Body:    []byte(expectedBody),
	})
	request := "GET /test HTTP/1.1\r\nHost: test.com\r\n\r\n"

	status, headers, body, err := runHandleConnectionTest(t, h, request)
	require.NoError(t, err)

	assert.Equal(t, "HTTP/1.1 200 OK", status)
	assert.Equal(t, "text/plain", headers["Content-Type"])
	assert.Equal(t, strconv.Itoa(len(expectedBody)), headers["Content-Length"])
	assert.Equal(t, "keep-alive", headers["Connection"])
	assert.Equal(t, expectedBody, string(body))
}

func TestHandleConnection_ValidPOST(t *testing.T) {
	requestBody := "posted data"
	h := func(ctx context.Context, req types.Request) types.Response {
		assert.NotNil(t, req.Body)
		assert.Equal(t, requestBody, *req.Body)
		return types.Response{
			Status:  types.StatusCreated,
			Headers: map[string]string{"Location": "/new-resource"},
			Body:    nil, // No body in response
		}
	}
	request := fmt.Sprintf("POST /submit HTTP/1.1\r\nHost: test.com\r\nContent-Length: %d\r\n\r\n%s", len(requestBody), requestBody)

	status, headers, body, err := runHandleConnectionTest(t, h, request)
	require.NoError(t, err)

	assert.Equal(t, "HTTP/1.1 201 Created", status)
	assert.Equal(t, "/new-resource", headers["Location"])
	assert.Equal(t, "0", headers["Content-Length"])
	assert.Equal(t, "keep-alive", headers["Connection"])
	assert.Empty(t, body)
}

func TestHandleConnection_ChunkedResponse(t *testing.T) {
	expectedBody := "Chunked response body."
	h := mockHandler(types.Response{
		Status:     types.StatusOK,
		Headers:    map[string]string{"X-Custom": "chunked-test"},
		BodyReader: strings.NewReader(expectedBody),
	})
	request := "GET /chunked HTTP/1.1\r\nHost: test.com\r\n\r\n"

	status, headers, body, err := runHandleConnectionTest(t, h, request)
	require.NoError(t, err)

	assert.Equal(t, "HTTP/1.1 200 OK", status)
	assert.Equal(t, "chunked", headers["Transfer-Encoding"])
	assert.NotContains(t, headers, "Content-Length")
	assert.Equal(t, "chunked-test", headers["X-Custom"])
	assert.Equal(t, "keep-alive", headers["Connection"])
	assert.Equal(t, expectedBody, string(body))
}

func TestHandleConnection_GzipResponse(t *testing.T) {
	originalBody := "This should be gzipped."
	h := mockHandler(types.Response{
		Status:  types.StatusOK,
		Headers: map[string]string{"Content-Type": "text/plain"},
		Body:    []byte(originalBody),
	})
	// Client requests gzip
	request := "GET /gzip HTTP/1.1\r\nHost: test.com\r\nAccept-Encoding: gzip\r\n\r\n"

	status, headers, body, err := runHandleConnectionTest(t, h, request)
	require.NoError(t, err)

	assert.Equal(t, "HTTP/1.1 200 OK", status)
	assert.Equal(t, "gzip", headers["Content-Encoding"])
	assert.NotEmpty(t, headers["Content-Length"], "Content-Length should be present for gzipped non-chunked")
	assert.Equal(t, "keep-alive", headers["Connection"])

	// Verify body is actually gzipped and decodes correctly
	gzReader, err := gzip.NewReader(bytes.NewReader(body))
	require.NoError(t, err)
	defer gzReader.Close()
	decodedBody, err := io.ReadAll(gzReader)
	require.NoError(t, err)
	assert.Equal(t, originalBody, string(decodedBody))
	assert.Equal(t, strconv.Itoa(len(body)), headers["Content-Length"], "Content-Length should match gzipped size")
}

func TestHandleConnection_MalformedRequestLine(t *testing.T) {
	h := func(ctx context.Context, req types.Request) types.Response {
		t.Error("Handler should not be called for malformed request")
		return types.Response{Status: types.StatusOK} // Should not be sent
	}
	request := "GET / HTTP/1.1 extra\r\nHost: test.com\r\n\r\n"

	serverConn, clientConn := net.Pipe()
	s := &Server{handler: h}
	go func() {
		defer serverConn.Close()
		s.handleConnection(serverConn)
	}()
	_, writeErr := clientConn.Write([]byte(request))
	require.NoError(t, writeErr)

	reader := bufio.NewReader(clientConn)
	statusLineBytes, readErr := reader.ReadBytes('\n')
	clientConn.Close()

	require.NoError(t, readErr, "Should have received at least the status line")
	statusLine := strings.TrimRight(string(statusLineBytes), "\r\n")
	assert.Equal(t, "HTTP/1.1 400 Bad Request", statusLine)
}

func TestHandleConnection_HandlerReturnsNotFound(t *testing.T) {
	h := mockHandler(types.Response{
		Status:  types.StatusNotFound,
		Headers: map[string]string{"Content-Type": "text/plain"},
		Body:    []byte("Resource Missing"),
	})
	request := "GET /not/found HTTP/1.1\r\nHost: test.com\r\n\r\n"

	status, headers, body, err := runHandleConnectionTest(t, h, request)
	require.NoError(t, err)

	assert.Equal(t, "HTTP/1.1 404 Not Found", status)
	assert.Equal(t, "close", headers["Connection"], "Connection should be close for 4xx errors")
	assert.Equal(t, "Resource Missing", string(body))
}

func TestHandleConnection_HandlerReturnsServerError(t *testing.T) {
	h := mockHandler(types.Response{
		Status: types.StatusInternalServerError,
	})
	request := "GET /error HTTP/1.1\r\nHost: test.com\r\n\r\n"

	status, headers, _, err := runHandleConnectionTest(t, h, request)
	require.NoError(t, err)

	assert.Equal(t, "HTTP/1.1 500 Internal Server Error", status)
	assert.Equal(t, "close", headers["Connection"], "Connection should be close for 5xx errors")
}

func TestHandleConnection_ClientRequestsClose(t *testing.T) {
	h := mockHandler(types.Response{Status: types.StatusOK, Body: []byte("OK")})
	request := "GET / HTTP/1.1\r\nHost: test.com\r\nConnection: close\r\n\r\n"

	status, headers, _, err := runHandleConnectionTest(t, h, request)
	require.NoError(t, err)

	assert.Equal(t, "HTTP/1.1 200 OK", status)
	assert.Equal(t, "close", headers["Connection"], "Server should honor Connection: close request")
}

func TestHandleConnection_ServerErrorForcesClose(t *testing.T) {
	h := mockHandler(types.Response{
		Status: types.StatusNotFound,
	})
	request := "GET /not/found HTTP/1.1\r\nHost: test.com\r\n\r\n"

	status, headers, _, err := runHandleConnectionTest(t, h, request)
	require.NoError(t, err)

	assert.Equal(t, "HTTP/1.1 404 Not Found", status)
	assert.Equal(t, "close", headers["Connection"])
}
