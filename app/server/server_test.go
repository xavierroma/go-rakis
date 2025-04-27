package server

import (
	"bytes"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/codecrafters-io/http-server-starter-go/app/types"
)

// TestParseRequest tests the parsing of various HTTP requests.
func TestParseRequest(t *testing.T) {
	tests := []struct {
		name        string
		request     string
		want        types.Request
		expectError bool
	}{
		{
			name:    "Valid GET Request",
			request: "GET /index.html HTTP/1.1\r\nHost: example.com\r\nUser-Agent: test\r\n\r\n",
			want: types.Request{
				Method:  types.Get,
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
			want: types.Request{
				Method:  types.Post,
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
			want: types.Request{ // Should still parse the request line correctly
				Method:  types.Get,
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
