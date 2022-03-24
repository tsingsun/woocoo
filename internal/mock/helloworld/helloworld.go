package helloworld

import (
	"context"
	"log"
)

// server is used to implement helloworld.GreeterServer.
type Server struct {
	UnimplementedGreeterServer
	count int
}

// SayHello implements helloworld.GreeterServer
func (s *Server) SayHello(ctx context.Context, in *HelloRequest) (*HelloReply, error) {
	s.count++
	log.Printf("Received %d: %v", s.count, in.GetName())
	return &HelloReply{Message: "Hello " + in.GetName()}, nil
}
