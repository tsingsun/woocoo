package interceptor_test

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	"github.com/tsingsun/woocoo/rpc/grpcx/interceptor"
	"github.com/tsingsun/woocoo/test"
	"github.com/tsingsun/woocoo/test/testdata"
	"github.com/tsingsun/woocoo/test/testproto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"net"
	"testing"
	"time"
)

func applog() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		log.AppendToIncomingContext(ctx, zap.String("logger_test", "test"))
		return handler(ctx, req)
	}
}

func TestGrpcContextLogger(t *testing.T) {
	logger := log.Component("grpc")
	logger.SetLogger(log.New(zap.NewExample()))
	addr := "localhost:50054"
	p := conf.NewParserFromStringMap(map[string]interface{}{
		"TimestampFormat": "2006-01-02 15:04:05",
	})
	cnf := conf.New(conf.WithLocalPath(testdata.TestConfigFile()), conf.WithBaseDir(testdata.BaseDir())).Load()
	clfg := cnf.CutFromParser(p)
	gloger := log.New(nil)
	assert.NotPanics(t, func() { gloger.Apply(cnf.Sub("log")) })
	gloger.AsGlobal()
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
	ctx := metadata.AppendToOutgoingContext(context.Background(), log.TraceID, "uuidtest")
	resp, err := client.Ping(ctx, &testproto.PingRequest{
		Value: t.Name(),
	})
	assert.NoError(t, err)
	assert.EqualValues(t, 42, resp.Counter)
}

func TestLoggerUnaryServerInterceptor(t *testing.T) {
	type fields struct {
		ctx     context.Context
		handler grpc.UnaryHandler
		info    *grpc.UnaryServerInfo
	}
	type args struct {
		cfg *conf.Configuration
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    func() *test.StringWriteSyncer
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "info",
			args: args{
				cfg: conf.NewFromStringMap(map[string]interface{}{}),
			},
			fields: fields{
				ctx: context.Background(),
				handler: func(ctx context.Context, req interface{}) (interface{}, error) {
					log.AppendToIncomingContext(ctx, zap.String("woocoo", "test"))
					return nil, nil
				},
				info: &grpc.UnaryServerInfo{FullMethod: "test"},
			},
			want: func() *test.StringWriteSyncer {
				logdata := &test.StringWriteSyncer{}
				log.New(test.NewStringLogger(logdata)).AsGlobal()
				return logdata
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				ss := i[0].(*test.StringWriteSyncer)
				assert.Len(t, ss.Entry, 1)
				assert.Contains(t, ss.Entry[0], "info")
				assert.Contains(t, ss.Entry[0], "woocoo")
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := tt.want()

			unaryServerInterceptor := interceptor.LoggerUnaryServerInterceptor(tt.args.cfg)
			got, err := unaryServerInterceptor(tt.fields.ctx, nil, tt.fields.info, tt.fields.handler)
			if !tt.wantErr(t, err, want) {
				return
			}
			assert.Nil(t, got)
		})
	}
}
