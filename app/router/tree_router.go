package router

import (
	"context"

	"github.com/codecrafters-io/http-server-starter-go/app/segmenttree"
	"github.com/codecrafters-io/http-server-starter-go/app/types"
)

type treeRouter struct {
	tree *segmenttree.SegmentTree
}

func newTreeRouter() *treeRouter {
	return &treeRouter{
		tree: segmenttree.NewSegmentTree(),
	}
}

func (r *treeRouter) Register(method types.Method, path string, handler types.Handler) Router {
	r.tree.Insert(method, path, handler)
	return r
}

func (r *treeRouter) HandleRequest(ctx context.Context, req types.Request) types.Response {
	handler, params, ok := r.tree.Search(req.Method, req.Target)
	if !ok {
		return types.Response{
			Status: types.StatusNotFound,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: []byte("404 Not Found"),
		}
	}

	req.Params = params
	response := types.Response{
		Status:  types.StatusOK,
		Headers: make(map[string]string),
	}

	handler(ctx, req, &response)
	return response
}
