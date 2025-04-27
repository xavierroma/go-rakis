package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/codecrafters-io/http-server-starter-go/app/server"
)

var directory string

func main() {
	flag.StringVar(&directory, "directory", "/tmp", "directory to serve files from")
	flag.Parse()

	fmt.Println("Logs from your program will appear here!")

	s := server.NewServer("0.0.0.0:4221")
	s.RegisterHandler(server.Get, "/", func(ctx context.Context, req server.Request, res *server.Response) {
		res.Status = server.StatusOK
		res.Body = []byte("Hello, World!")
		res.Headers["Content-Type"] = "text/plain"
		res.Headers["Content-Length"] = fmt.Sprintf("%d", len(res.Body))
	})
	s.RegisterHandler(server.Get, "/echo/:path", func(ctx context.Context, req server.Request, res *server.Response) {
		res.Status = server.StatusOK
		res.Body = []byte(req.Params["path"])
		res.Headers["Content-Type"] = "text/plain"
		res.Headers["Content-Length"] = fmt.Sprintf("%d", len(res.Body))
	})
	s.RegisterHandler(server.Get, "/user-agent", func(ctx context.Context, req server.Request, res *server.Response) {
		res.Status = server.StatusOK
		res.Body = []byte(req.Headers["User-Agent"])
		res.Headers["Content-Type"] = "text/plain"
		res.Headers["Content-Length"] = fmt.Sprintf("%d", len(res.Body))
	})
	s.RegisterHandler(server.Get, "/files/:path", func(ctx context.Context, req server.Request, res *server.Response) {
		res.Status = server.StatusOK
		res.Body = []byte(req.Params["path"])
		res.Headers["Content-Type"] = "text/plain"
		res.Headers["Content-Length"] = fmt.Sprintf("%d", len(res.Body))
	})
	s.RegisterHandler(server.Post, "/files/:path", func(ctx context.Context, req server.Request, res *server.Response) {
		res.Status = server.StatusCreated
		res.Body = []byte(req.Params["path"])
		res.Headers["Content-Type"] = "text/plain"
		res.Headers["Content-Length"] = fmt.Sprintf("%d", len(res.Body))
	})
	s.Listen()
}
