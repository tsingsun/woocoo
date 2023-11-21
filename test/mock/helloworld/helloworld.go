package helloworld

import (
	"context"
)

// Server is used to implement helloworld.GreeterServer.
type Server struct {
	UnimplementedGreeterServer
	count int
	Tag   string
}

// SayHello implements helloworld.GreeterServer
func (s *Server) SayHello(ctx context.Context, in *HelloRequest) (*HelloReply, error) {
	s.count++
	if s.Tag != "" {
		s.Tag += " "
	}
	return &HelloReply{Message: s.Tag + "Hello " + in.GetName()}, nil
}
