package server

import (
	"bytes"
	"context"
	"net"
	"reflect"
	"testing"
	"time"
)

// TestParseRequest tests the parsing of various HTTP requests.
func TestParseRequest(t *testing.T) {
	tests := []struct {
		name        string
		request     string
		want        Request
		expectError bool
	}{
		{
			name:    "Valid GET Request",
			request: "GET /index.html HTTP/1.1\r\nHost: example.com\r\nUser-Agent: test\r\n\r\n",
			want: Request{
				Method:  Get,
				Target:  "/index.html",
				Version: "HTTP/1.1",
				Headers: map[string]string{"Host": "example.com", "User-Agent": "test"},
				Body:    nil,
			},
			expectError: false,
		},
		{
			name:    "Valid POST Request",
			request: "POST /submit HTTP/1.1\r\nHost: example.com\r\nContent-Length: 11\r\n\r\nHello World",
			want: Request{
				Method:  Post,
				Target:  "/submit",
				Version: "HTTP/1.1",
				Headers: map[string]string{"Host": "example.com", "Content-Length": "11"},
				Body:    func() *string { s := "Hello World"; return &s }(),
			},
			expectError: false,
		},
		{
			name:        "Malformed Request Line - Too Few Parts",
			request:     "GET /\r\nHost: example.com\r\n\r\n",
			expectError: true,
		},
		{
			name:        "Malformed Request Line - Empty",
			request:     "\r\n",
			expectError: true,
		},
		{
			name:    "Malformed Header Line",
			request: "GET / HTTP/1.1\r\nHost example.com\r\n\r\n", // Missing colon
			want: Request{ // Should still parse the request line correctly
				Method:  Get,
				Target:  "/",
				Version: "HTTP/1.1",
				Headers: map[string]string{}, // Malformed header is skipped
				Body:    nil,
			},
			expectError: false, // The current implementation logs a warning but doesn't return an error
		},
		{
			name:        "POST Request Invalid Content-Length",
			request:     "POST /submit HTTP/1.1\r\nHost: example.com\r\nContent-Length: abc\r\n\r\nHello World",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := newMockConn(tt.request)
			got, err := parseRequest(conn)

			if (err != nil) != tt.expectError {
				t.Errorf("parseRequest() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if !tt.expectError {
				// Compare fields individually for better error messages
				if got.Method != tt.want.Method {
					t.Errorf("parseRequest() got Method = %v, want %v", got.Method, tt.want.Method)
				}
				if got.Target != tt.want.Target {
					t.Errorf("parseRequest() got Target = %v, want %v", got.Target, tt.want.Target)
				}
				if got.Version != tt.want.Version {
					t.Errorf("parseRequest() got Version = %v, want %v", got.Version, tt.want.Version)
				}
				if !reflect.DeepEqual(got.Headers, tt.want.Headers) {
					t.Errorf("parseRequest() got Headers = %v, want %v", got.Headers, tt.want.Headers)
				}
				// Compare body content pointers if both are non-nil
				if (got.Body != nil && tt.want.Body != nil && *got.Body != *tt.want.Body) || (got.Body != nil) != (tt.want.Body != nil) {
					gotBodyStr := "<nil>"
					wantBodyStr := "<nil>"
					if got.Body != nil {
						gotBodyStr = *got.Body
					}
					if tt.want.Body != nil {
						wantBodyStr = *tt.want.Body
					}
					t.Errorf("parseRequest() got Body = %q, want %q", gotBodyStr, wantBodyStr)
				}
			}
		})
	}
}

// TestFindHandler tests the route matching logic with parameters.
func TestFindHandler(t *testing.T) {
	dummyHandler := func(ctx context.Context, req Request, res *Response) {}
	handlers := map[string]Handler{
		"/":              dummyHandler,
		"/users/:id":     dummyHandler,
		"/posts/:postId": dummyHandler,
		"/files/:file":   dummyHandler,
	}

	tests := []struct {
		name       string
		target     string
		wantParams map[string]string
		wantFound  bool
	}{
		{
			name:       "Exact Match Root",
			target:     "/",
			wantParams: map[string]string{},
			wantFound:  true,
		},
		{
			name:       "Single Param Match",
			target:     "/users/123",
			wantParams: map[string]string{"id": "123"},
			wantFound:  true,
		},
		{
			name:       "Different Param Match",
			target:     "/posts/abc",
			wantParams: map[string]string{"postId": "abc"},
			wantFound:  true,
		},
		{
			name:       "File Param Match",
			target:     "/files/document.txt",
			wantParams: map[string]string{"file": "document.txt"},
			wantFound:  true,
		},
		{
			name:       "No Match - Wrong Path",
			target:     "/products/456",
			wantParams: nil,
			wantFound:  false,
		},
		{
			name:       "No Match - Too Many Segments",
			target:     "/users/123/profile",
			wantParams: nil,
			wantFound:  false,
		},
		{
			name:       "No Match - Too Few Segments",
			target:     "/users",
			wantParams: nil,
			wantFound:  false,
		},
		{
			name:       "Match with Empty Segment", // e.g., /files//test -> should not match /files/:file
			target:     "/files//test",
			wantParams: nil,
			wantFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := Request{Target: tt.target} // Only target is needed for findHandler
			handler, found := findHandler(handlers, &req)

			if found != tt.wantFound {
				t.Errorf("findHandler() found = %v, wantFound %v", found, tt.wantFound)
			}

			if tt.wantFound {
				if handler == nil {
					t.Errorf("findHandler() expected handler to be non-nil when found is true")
				}
				if !reflect.DeepEqual(req.Params, tt.wantParams) {
					t.Errorf("findHandler() req.Params = %v, wantParams %v", req.Params, tt.wantParams)
				}
			} else {
				if handler != nil {
					t.Errorf("findHandler() expected handler to be nil when found is false")
				}
				// Check if Params map was potentially modified even if not found
				if len(req.Params) > 0 {
					t.Errorf("findHandler() req.Params should be empty when handler is not found, got %v", req.Params)
				}
			}
		})
	}
}

// --- Helper for TestParseRequest ---

// mockConn simulates a network connection for testing reading requests.
type mockConn struct {
	*bytes.Reader
	closed bool
}

func newMockConn(data string) *mockConn {
	return &mockConn{Reader: bytes.NewReader([]byte(data))}
}

// Implement net.Conn interface methods needed by parseRequest

func (m *mockConn) Read(b []byte) (n int, err error) {
	return m.Reader.Read(b)
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	// Not needed for parsing, but needed to satisfy net.Conn
	return len(b), nil
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080}
}

func (m *mockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(192, 0, 2, 1), Port: 12345}
}

func (m *mockConn) SetDeadline(t time.Time) error {
	return nil // No-op
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	return nil // No-op
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return nil // No-op
}
