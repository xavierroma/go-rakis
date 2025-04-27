package router

import (
	"context"

	"github.com/codecrafters-io/http-server-starter-go/app/types"
)

type Router interface {
	Register(method types.Method, path string, handler types.Handler) Router

	HandleRequest(ctx context.Context, req types.Request) types.Response
}

func New() Router {
	return newTreeRouter()
}
