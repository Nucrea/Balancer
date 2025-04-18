package backend

import (
	"context"
	"time"
)

type Item interface {
	Backend
	Id() string
	SetStatus(status Status)
	Status() (Status, time.Time)
	Connections() int
}

type Backend interface {
	Invoke(ctx context.Context, req Request) (Response, error)
	Health(ctx context.Context) error
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
