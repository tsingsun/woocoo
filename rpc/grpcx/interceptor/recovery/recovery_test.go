package recovery_test

import (
	"context"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/rpc/grpcx/interceptor/logger"
	"github.com/tsingsun/woocoo/rpc/grpcx/interceptor/recovery"
	"github.com/tsingsun/woocoo/test/testdata"
	"github.com/tsingsun/woocoo/test/testproto"
	"google.golang.org/grpc"
	"net"
	"testing"
	"time"
)

var (
	cnf  = conf.New(conf.LocalPath(testdata.TestConfigFile()), conf.BaseDir(testdata.BaseDir())).Load()
	addr = "localhost:50053"
)

func TestUnaryServerInterceptor(t *testing.T) {
	p := conf.NewParserFromStringMap(map[string]interface{}{
		"TimestampFormat": "2006-01-02 15:04:05",
	})
	clfg := conf.NewFromParse(p)
	gloger := log.Logger{}
	gloger.Apply(cnf.Sub("log"))
	go func() {
		opts := []grpc.ServerOption{
			// The following grpc.ServerOption adds an interceptor for all unary
			// RPCs. To configure an interceptor for streaming RPCs, see:
			// https://godoc.org/google.golang.org/grpc#StreamInterceptor
			grpc.ChainUnaryInterceptor(logger.UnaryServerInterceptor(clfg), recovery.UnaryServerInterceptor(cnf)),
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

	copts := []grpc.DialOption{}
	copts = append(copts, grpc.WithBlock(), grpc.WithInsecure())
	conn, err := grpc.Dial(addr, copts...)
	if err != nil {
		t.Fatal(err)
	}
	client := testproto.NewTestServiceClient(conn)
	//ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	//defer cancel()
	_, err = client.PingPanic(context.Background(), &testproto.PingRequest{
		Value: t.Name(),
	})
	if err == nil {
		t.Error("must error")
	}
	gloger.Sync()
}
