package server

import (
	"context"
)

func NewServer(ctx context.Context, one serviceOne, two serviceTwo) *Server {
	return &Server{}
}

type serviceOne interface {
	MethodOne()
}

type serviceTwo interface {
	MethodTwo()
}

type Server struct {
}

func (s *Server) Serve() error {
	return nil
}
