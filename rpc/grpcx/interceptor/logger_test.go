package interceptor_test

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	"github.com/tsingsun/woocoo/rpc/grpcx/interceptor"
	"github.com/tsingsun/woocoo/test/testdata"
	"github.com/tsingsun/woocoo/test/testproto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net"
	"testing"
	"time"
)

var (
	cnf = conf.New(conf.WithLocalPath(testdata.TestConfigFile()), conf.WithBaseDir(testdata.BaseDir())).Load()
)

func applog() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		if lctx, ok := interceptor.LoggerFromIncomingContext(ctx); ok {
			newctx := interceptor.AppendLoggerToContext(ctx, lctx, zap.String("logger_test", "test"))
			return handler(newctx, req)
		}
		return handler(ctx, req)
	}
}

func TestUnaryServerInterceptor(t *testing.T) {
	addr := "localhost:50052"
	p := conf.NewParserFromStringMap(map[string]interface{}{
		"TimestampFormat": "2006-01-02 15:04:05",
	})
	clfg := cnf.CutFromParser(p)
	gloger := log.Logger{}
	assert.NotPanics(t, func() { gloger.Apply(cnf.Sub("log")) })
	go func() {
		opts := []grpc.ServerOption{
			// The following grpc.ServerOption adds an interceptor for all unary
			// RPCs. To configure an interceptor for streaming RPCs, see:
			// https://godoc.org/google.golang.org/grpc#StreamInterceptor
			grpc.ChainUnaryInterceptor(interceptor.LoggerUnaryServerInterceptor(clfg), applog()),
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
	time.Sleep(1000)

	var copts []grpc.DialOption
	copts = append(copts, grpc.WithBlock(), grpc.WithInsecure())
	conn, err := grpc.Dial(addr, copts...)
	if err != nil {
		t.Fatal(err)
	}
	client := testproto.NewTestServiceClient(conn)
	//ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	//defer cancel()
	resp, err := client.Ping(context.Background(), &testproto.PingRequest{
		Value: t.Name(),
	})
	if err != nil {
		t.Error(err)
	}
	t.Log(resp)
}

func TestGrpcContextLogger(t *testing.T) {
	addr := "localhost:50054"
	p := conf.NewParserFromStringMap(map[string]interface{}{
		"TimestampFormat": "2006-01-02 15:04:05",
	})
	clfg := cnf.CutFromParser(p)
	ss := make(chan bool)
	go func() {
		opts := []grpc.ServerOption{
			// The following grpc.ServerOption adds an interceptor for all unary
			// RPCs. To configure an interceptor for streaming RPCs, see:
			// https://godoc.org/google.golang.org/grpc#StreamInterceptor
			grpc.ChainUnaryInterceptor(interceptor.LoggerUnaryServerInterceptor(clfg), applog()),
		}
		gx := grpcx.New(grpcx.UseLogger(), grpcx.WithGrpcOption(opts...))
		log.Component("grpc").Logger().WithTraceID = true
		s := gx.Engine()
		testproto.RegisterTestServiceServer(s, &testproto.TestPingService{})
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			t.Errorf("failed to listen: %v", err)
			return
		}
		ss <- true
		if err := s.Serve(lis); err != nil {
			t.Errorf("failed to serve: %v", err)
			return
		}
	}()
	<-ss
	time.Sleep(1000)

	var copts []grpc.DialOption
	copts = append(copts, grpc.WithBlock(), grpc.WithInsecure())
	conn, err := grpc.Dial(addr, copts...)
	if err != nil {
		t.Fatal(err)
	}
	client := testproto.NewTestServiceClient(conn)
	//ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	//defer cancel()
	resp, err := client.Ping(context.Background(), &testproto.PingRequest{
		Value: t.Name(),
	})
	assert.NoError(t, err)
	t.Log(resp)
}
