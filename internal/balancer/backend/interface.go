package backend

import (
	"context"
	"time"
)

type Item interface {
	Id() string
	SetStatus(status Status)
	Status() (Status, time.Time)
	Connections() int
	Invoke(ctx context.Context, req Request) (Response, error)
}

type Backend interface {
	Invoke(ctx context.Context, req Request) (Response, error)
}

type Request struct {
	Method  string
	Path    string
	Body    []byte
	Headers map[string]string
}

type Response struct {
	Status  int
	Body    []byte
	Headers map[string]string
}
