package types

import (
	"context"
	"io"
)

type Method string

const (
	Get    Method = "GET"
	Post   Method = "POST"
	Put    Method = "PUT"
	Patch  Method = "PATCH"
	Delete Method = "DELETE"
)

type Handler func(ctx context.Context, req Request, res *Response)

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
	Status     Status
	Body       []byte
	BodyReader io.Reader
	Headers    map[string]string
}
