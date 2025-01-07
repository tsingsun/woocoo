package otelgrpc

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	"github.com/tsingsun/woocoo/test/mock/helloworld"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"io"
	"log"
	"net"
	"testing"
	"time"
)

const target = "passthrough://bufnet"

func initTracer(writer io.Writer) func() {
	// Create stdout exporter
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint(), stdouttrace.WithWriter(writer))
	if err != nil {
		log.Fatalf("failed to initialize stdouttrace exporter: %v", err)
	}

	// Create trace provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)

	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatalf("failed to shutdown TracerProvider: %v", err)
		}
	}
}

func TestOTELGrpc(t *testing.T) {
	var buf bytes.Buffer
	cleanup := initTracer(&buf)
	grpcx.RegisterGrpcUnaryInterceptor("trace", func(configuration *conf.Configuration) grpc.UnaryServerInterceptor {
		return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
			return handler(ctx, req)
		}
	})
	lis := bufconn.Listen(1024)
	srv := grpcx.New(grpcx.WithConfiguration(conf.NewFromBytes([]byte(`
server:
  addr: 127.0.0.1:0
engine:
  - otel:
  - unaryInterceptors:
    - trace:
`))), grpcx.WithListener(lis),
	)
	helloworld.RegisterGreeterServer(srv.Engine(), &helloworld.Server{})

	go func() {
		if err := srv.Start(context.Background()); err != nil {
			t.Fail()
		}
	}()
	srv.Engine()
	client, err := grpcx.NewClient(conf.NewFromBytes([]byte(`
server:
  addr: 127.0.0.1:0
client:
  dialOption:
  - otel:
`)))
	conn, err := client.Dial(target, grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return lis.Dial()
	}))
	require.NoError(t, err)

	hcli := helloworld.NewGreeterClient(conn)
	resp, err := hcli.SayHello(context.Background(), &helloworld.HelloRequest{Name: "test"})
	require.NoError(t, err)
	require.Equal(t, "Hello test", resp.Message)
	// make output
	cleanup()
	assert.Contains(t, buf.String(), "helloworld.Greeter/SayHello")
	select {
	case <-time.After(1 * time.Second):
		srv.Stop(context.Background())
	}
}
