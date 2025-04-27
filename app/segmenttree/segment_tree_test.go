package segmenttree

import (
	"context"
	"reflect"
	"testing"

	"github.com/codecrafters-io/http-server-starter-go/app/types"
)

func TestSegmentTreeRouting(t *testing.T) {
	type testRoute struct {
		method  types.Method
		path    string
		handler types.Handler
	}

	type testCase struct {
		name          string
		routes        []testRoute
		searchMethod  types.Method
		searchPath    string
		wantMatch     bool
		wantParams    map[string]string
		wantHandlerID int // index into routes slice for expected handler
	}

	// Create unique handlers that we can identify later
	handlers := make([]types.Handler, 10)
	for i := range handlers {
		i := i // capture loop variable
		handlers[i] = func(ctx context.Context, req types.Request, res *types.Response) {}
	}

	tests := []testCase{
		{
			name: "Exact static routes",
			routes: []testRoute{
				{types.Get, "/about", handlers[0]},
				{types.Get, "/foo/bar", handlers[1]},
			},
			searchMethod:  types.Get,
			searchPath:    "/foo/bar",
			wantMatch:     true,
			wantParams:    map[string]string{},
			wantHandlerID: 1,
		},
		{
			name: "Root path",
			routes: []testRoute{
				{types.Get, "/", handlers[0]},
			},
			searchMethod:  types.Get,
			searchPath:    "/",
			wantMatch:     true,
			wantParams:    map[string]string{},
			wantHandlerID: 0,
		},
		{
			name: "Not found case",
			routes: []testRoute{
				{types.Get, "/exists", handlers[0]},
			},
			searchMethod: types.Get,
			searchPath:   "/does-not-exist",
			wantMatch:    false,
			wantParams:   nil,
		},
		{
			name: "Trailing slash mismatch",
			routes: []testRoute{
				{types.Get, "/foo", handlers[0]},
			},
			searchMethod: types.Get,
			searchPath:   "/foo/",
			wantMatch:    false,
			wantParams:   nil,
		},
		{
			name: "Single param capture",
			routes: []testRoute{
				{types.Get, "/users/:id", handlers[0]},
			},
			searchMethod:  types.Get,
			searchPath:    "/users/42",
			wantMatch:     true,
			wantParams:    map[string]string{"id": "42"},
			wantHandlerID: 0,
		},
		{
			name: "Multiple params in one path",
			routes: []testRoute{
				{types.Get, "/orders/:orderId/items/:itemId", handlers[0]},
			},
			searchMethod:  types.Get,
			searchPath:    "/orders/123/items/456",
			wantMatch:     true,
			wantParams:    map[string]string{"orderId": "123", "itemId": "456"},
			wantHandlerID: 0,
		},
		{
			name: "Multiple params with same segment position but different names",
			routes: []testRoute{
				{types.Get, "/echo/:msg/v1", handlers[0]},
				{types.Get, "/echo/:message/v2", handlers[1]},
			},
			searchMethod:  types.Get,
			searchPath:    "/echo/hello/v1",
			wantMatch:     true,
			wantParams:    map[string]string{"msg": "hello"},
			wantHandlerID: 0,
		},
		{
			name: "Multiple params with same segment position - second route",
			routes: []testRoute{
				{types.Get, "/echo/:msg/v1", handlers[0]},
				{types.Get, "/echo/:message/v2", handlers[1]},
			},
			searchMethod:  types.Get,
			searchPath:    "/echo/world/v2",
			wantMatch:     true,
			wantParams:    map[string]string{"message": "world"},
			wantHandlerID: 1,
		},
		{
			name: "Empty segment rejection",
			routes: []testRoute{
				{types.Get, "/files/:file", handlers[0]},
			},
			searchMethod: types.Get,
			searchPath:   "/files//foo",
			wantMatch:    false,
			wantParams:   nil,
		},
		{
			name: "Different HTTP methods on same path",
			routes: []testRoute{
				{types.Get, "/foo", handlers[0]},
				{types.Post, "/foo", handlers[1]},
			},
			searchMethod:  types.Post,
			searchPath:    "/foo",
			wantMatch:     true,
			wantParams:    map[string]string{},
			wantHandlerID: 1,
		},
		{
			name: "Re-inserting same route",
			routes: []testRoute{
				{types.Get, "/x/:id", handlers[0]},
				{types.Get, "/x/:id", handlers[1]}, // Should override
			},
			searchMethod:  types.Get,
			searchPath:    "/x/1",
			wantMatch:     true,
			wantParams:    map[string]string{"id": "1"},
			wantHandlerID: 1,
		},
		{
			name: "Param value character edge cases",
			routes: []testRoute{
				{types.Get, "/assets/:name", handlers[0]},
			},
			searchMethod:  types.Get,
			searchPath:    "/assets/logo-v2.png",
			wantMatch:     true,
			wantParams:    map[string]string{"name": "logo-v2.png"},
			wantHandlerID: 0,
		},
		{
			name: "Static precedence over param - exact match",
			routes: []testRoute{
				{types.Get, "/items/:id", handlers[0]},
				{types.Get, "/items/special", handlers[1]},
			},
			searchMethod:  types.Get,
			searchPath:    "/items/special",
			wantMatch:     true,
			wantParams:    map[string]string{},
			wantHandlerID: 1,
		},
		{
			name: "Static precedence over param - param fallback",
			routes: []testRoute{
				{types.Get, "/items/:id", handlers[0]},
				{types.Get, "/items/special", handlers[1]},
			},
			searchMethod:  types.Get,
			searchPath:    "/items/123",
			wantMatch:     true,
			wantParams:    map[string]string{"id": "123"},
			wantHandlerID: 0,
		},
		{
			name: "Deeply nested static vs param conflicts - static wins",
			routes: []testRoute{
				{types.Get, "/a/b/c", handlers[0]},
				{types.Get, "/a/:b/c", handlers[1]},
			},
			searchMethod:  types.Get,
			searchPath:    "/a/b/c",
			wantMatch:     true,
			wantParams:    map[string]string{},
			wantHandlerID: 0,
		},
		{
			name: "Deeply nested static vs param conflicts - param captures",
			routes: []testRoute{
				{types.Get, "/a/b/c", handlers[0]},
				{types.Get, "/a/:b/c", handlers[1]},
			},
			searchMethod:  types.Get,
			searchPath:    "/a/xyz/c",
			wantMatch:     true,
			wantParams:    map[string]string{"b": "xyz"},
			wantHandlerID: 1,
		},
		{
			name: "Repeated param names at different depths",
			routes: []testRoute{
				{types.Get, "/foo/:id/bar/:id", handlers[0]},
			},
			searchMethod:  types.Get,
			searchPath:    "/foo/123/bar/456",
			wantMatch:     true,
			wantParams:    map[string]string{"id": "456"}, // Last value wins
			wantHandlerID: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewSegmentTree()
			for _, r := range tt.routes {
				tr.Insert(r.method, r.path, r.handler)
			}

			gotHandler, gotParams, gotOk := tr.Search(tt.searchMethod, tt.searchPath)

			if tt.wantMatch != gotOk {
				t.Errorf("match = %v, want %v", gotOk, tt.wantMatch)
				return
			}

			if !tt.wantMatch {
				if gotHandler != nil {
					t.Error("handler = non-nil, want nil for no-match")
				}
				return
			}

			// Compare handler by checking if it's the one we expect from our handlers slice
			gotHandlerPtr := reflect.ValueOf(gotHandler).Pointer()
			wantHandlerPtr := reflect.ValueOf(handlers[tt.wantHandlerID]).Pointer()
			if gotHandlerPtr != wantHandlerPtr {
				t.Errorf("got handler %v, want handler[%d]", gotHandler, tt.wantHandlerID)
			}

			if !reflect.DeepEqual(gotParams, tt.wantParams) {
				t.Errorf("params = %v, want %v", gotParams, tt.wantParams)
			}
		})
	}
}
