package interceptor

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/test"
	"github.com/tsingsun/woocoo/test/testproto"
	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"net"
	"testing"
	"time"
)

func TestRecoveryUnaryServerInterceptor(t *testing.T) {
	addr := "localhost:50053"
	logdata := test.NewGlobalStringLogger()

	zgl := zapgrpc.NewLogger(log.Global().Operator())
	grpclog.SetLoggerV2(zgl)
	go func() {
		clfg := conf.NewFromStringMap(map[string]interface{}{
			"TimestampFormat": "2006-01-02 15:04:05",
		})
		cnf := conf.NewFromStringMap(map[string]interface{}{})
		opts := []grpc.ServerOption{
			// The following grpc.ServerOption adds an interceptor for all unary
			// RPCs. To configure an interceptor for streaming RPCs, see:
			// https://godoc.org/google.golang.org/grpc#StreamInterceptor
			grpc.ChainUnaryInterceptor(LoggerUnaryServerInterceptor(clfg), RecoveryUnaryServerInterceptor(cnf)),
		}

		s := grpc.NewServer(opts...)
		testproto.RegisterTestServiceServer(s, &testproto.TestPingService{})
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			t.Errorf("failed to listen: %v", err)
			return
		}
		if err := s.Serve(lis); err != nil {
			t.Errorf("failed to serve: %v", err)
			return
		}
	}()
	time.Sleep(time.Second)

	var copts []grpc.DialOption
	copts = append(copts, grpc.WithBlock(), grpc.WithInsecure())
	conn, err := grpc.Dial(addr, copts...)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("stacktrace", func(t *testing.T) {
		log.Global().DisableStacktrace = false
		client := testproto.NewTestServiceClient(conn)
		_, err = client.PingPanic(context.Background(), &testproto.PingRequest{
			Value: t.Name(),
		})
		require.Error(t, err)
		last := logdata.Entry[len(logdata.Entry)-1]
		require.Contains(t, last, "grpc panic error")
		require.Contains(t, last, "testproto/pingservice.go")
	})
	t.Run("disableStacktrace", func(t *testing.T) {
		log.Global().DisableStacktrace = true
		client := testproto.NewTestServiceClient(conn)
		_, err = client.PingPanic(context.Background(), &testproto.PingRequest{
			Value: t.Name(),
		})
		require.Error(t, err)
		last := logdata.Entry[len(logdata.Entry)-1]
		require.Contains(t, last, "grpc panic error")
		require.Contains(t, last, "testproto/pingservice.go")
	})
}

func TestRecoveryUnaryServerInterceptorWithoutLogger(t *testing.T) {
	addr := "localhost:50055"
	logdata := test.NewGlobalStringLogger()

	zgl := zapgrpc.NewLogger(log.Global().Operator())
	grpclog.SetLoggerV2(zgl)
	go func() {
		cnf := conf.NewFromStringMap(map[string]interface{}{})
		opts := []grpc.ServerOption{
			grpc.ChainUnaryInterceptor(RecoveryUnaryServerInterceptor(cnf)),
		}

		s := grpc.NewServer(opts...)
		testproto.RegisterTestServiceServer(s, &testproto.TestPingService{})
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			t.Errorf("failed to listen: %v", err)
			return
		}
		if err := s.Serve(lis); err != nil {
			t.Errorf("failed to serve: %v", err)
			return
		}
	}()
	time.Sleep(time.Second)

	var copts []grpc.DialOption
	copts = append(copts, grpc.WithBlock(), grpc.WithInsecure())
	conn, err := grpc.Dial(addr, copts...)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("stacktrace", func(t *testing.T) {
		log.Global().DisableStacktrace = false
		client := testproto.NewTestServiceClient(conn)
		_, err = client.PingPanic(context.Background(), &testproto.PingRequest{
			Value: t.Name(),
		})
		require.Error(t, err)
		last := logdata.Entry[len(logdata.Entry)-1]
		require.Contains(t, last, "grpc panic error")
		require.Contains(t, last, "testproto/pingservice.go")
	})
	t.Run("disableStacktrace", func(t *testing.T) {
		log.Global().DisableStacktrace = true
		client := testproto.NewTestServiceClient(conn)
		_, err = client.PingPanic(context.Background(), &testproto.PingRequest{
			Value: t.Name(),
		})
		require.Error(t, err)
		last := logdata.Entry[len(logdata.Entry)-1]
		require.Contains(t, last, "[Recovery from panic]")
		require.Contains(t, last, "testproto/pingservice.go")
	})
}
