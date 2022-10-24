package testproto

import (
	"context"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"io"
	"log"
	"net"
	"testing"
	"time"

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

func NewPingGrpcService(t *testing.T, opts ...grpc.ServerOption) (server *grpc.Server, addr string) {
	addr = "localhost:50053"
	ch := make(chan struct{})
	go func() {
		server = grpc.NewServer(opts...)
		RegisterTestServiceServer(server, &TestPingService{})
		ch <- struct{}{}
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			t.Errorf("failed to listen: %v", err)
			return
		}
		if err := server.Serve(lis); err != nil {
			t.Errorf("failed to serve: %v", err)
			return
		}
	}()
	if <-ch; true {
		time.Sleep(time.Microsecond * 100)
	}
	return
}

func NewPingGrpcClient(t *testing.T, ctx context.Context, addr string, opts ...grpc.DialOption) (conn *grpc.ClientConn, client TestServiceClient) {
	var copts []grpc.DialOption
	if len(opts) == 0 {
		copts = []grpc.DialOption{grpc.WithBlock(), grpc.WithInsecure()}
	} else {
		copts = opts
	}
	conn, err := grpc.DialContext(ctx, addr, copts...)
	require.NoError(t, err)
	client = NewTestServiceClient(conn)
	return
}
