package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/codecrafters-io/http-server-starter-go/app/router"
	"github.com/codecrafters-io/http-server-starter-go/app/server"
	"github.com/codecrafters-io/http-server-starter-go/app/types"
)

var directory string

func main() {
	flag.StringVar(&directory, "directory", "/tmp", "directory to serve files from")
	flag.Parse()

	fmt.Println("Logs from your program will appear here!")

	r := router.New()

	r.Register(types.Get, "/", func(ctx context.Context, req types.Request, res *types.Response) {
		res.Status = types.StatusOK
		res.Body = []byte("Hello, World!")
		res.Headers["Content-Type"] = "text/plain"
		res.Headers["Content-Length"] = fmt.Sprintf("%d", len(res.Body))
	})

	r.Register(types.Get, "/echo/:path", func(ctx context.Context, req types.Request, res *types.Response) {
		res.Status = types.StatusOK
		res.Body = []byte(req.Params["path"])
		res.Headers["Content-Type"] = "text/plain"
		res.Headers["Content-Length"] = fmt.Sprintf("%d", len(res.Body))
	})

	r.Register(types.Get, "/user-agent", func(ctx context.Context, req types.Request, res *types.Response) {
		res.Status = types.StatusOK
		res.Body = []byte(req.Headers["User-Agent"])
		res.Headers["Content-Type"] = "text/plain"
		res.Headers["Content-Length"] = fmt.Sprintf("%d", len(res.Body))
	})

	r.Register(types.Get, "/files/:path", func(ctx context.Context, req types.Request, res *types.Response) {
		res.Status = types.StatusOK
		res.Body = []byte(req.Params["path"])
		res.Headers["Content-Type"] = "text/plain"
		res.Headers["Content-Length"] = fmt.Sprintf("%d", len(res.Body))
	})

	r.Register(types.Post, "/files/:path", func(ctx context.Context, req types.Request, res *types.Response) {
		res.Status = types.StatusCreated
		res.Body = []byte(req.Params["path"])
		res.Headers["Content-Type"] = "text/plain"
		res.Headers["Content-Length"] = fmt.Sprintf("%d", len(res.Body))
	})

	s := server.NewServer("0.0.0.0:4221").WithHandler(r.HandleRequest)
	s.Listen()
}
