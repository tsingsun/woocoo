// Copyright 2016 Michal Witkowski. All Rights Reserved.
// See LICENSE for licensing terms.

/*
Package `grpc_testing` provides helper functions for testing validators in this package.
*/

package testproto

import (
	"context"
	"io"
	"log"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// DefaultResponseValue is the default value used.
	DefaultResponseValue = "default_response_value"
	// ListResponseCount is the expected number of responses to PingList
	ListResponseCount = 100
)

type TestPingService struct {
	T *testing.T
	UnimplementedTestServiceServer
}

func (s *TestPingService) PingEmpty(ctx context.Context, _ *Empty) (*PingResponse, error) {
	return &PingResponse{Value: DefaultResponseValue, Counter: 42}, nil
}

func (s *TestPingService) Ping(ctx context.Context, ping *PingRequest) (*PingResponse, error) {
	// Send user trailers and headers.
	log.Printf("Received %s", ping.Value)
	return &PingResponse{Value: ping.Value, Counter: 42}, nil
}

func (s *TestPingService) PingError(ctx context.Context, ping *PingRequest) (*Empty, error) {
	code := codes.Code(ping.ErrorCodeReturned)
	return nil, status.Errorf(code, "Userspace error.")
}

func (s *TestPingService) PingPanic(ctx context.Context, ping *PingRequest) (*Empty, error) {
	panic("grpc panic error")
}

func (s *TestPingService) PingList(ping *PingRequest, stream TestService_PingListServer) error {
	if ping.ErrorCodeReturned != 0 {
		return status.Errorf(codes.Code(ping.ErrorCodeReturned), "foobar")
	}
	// Send user trailers and headers.
	for i := 0; i < ListResponseCount; i++ {
		err := stream.Send(&PingResponse{Value: ping.Value, Counter: int32(i)})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *TestPingService) PingStream(stream TestService_PingStreamServer) error {
	count := 0
	for {
		ping, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		stream.Send(&PingResponse{Value: ping.Value, Counter: int32(count)})
		count += 1
	}
	return nil
}
