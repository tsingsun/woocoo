package interceptor

import (
	"context"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/codes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/test/logtest"
	"github.com/tsingsun/woocoo/test/testproto"
	"github.com/tsingsun/woocoo/test/wctest"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func applog() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		log.AppendToIncomingContext(ctx, zap.String("logger_test", "test"))
		return handler(ctx, req)
	}
}

func TestGrpcContextLogger(t *testing.T) {
	logdata := wctest.InitBuffWriteSyncer(zap.AddStacktrace(zap.ErrorLevel))
	// get the global logger
	log.Component(log.GrpcComponentName).Logger().SetContextLogger(NewGrpcContextLogger())
	log.Component(log.GrpcComponentName).Logger(log.WithOriginalLogger()).WithTraceID = true
	log.Component(log.GrpcComponentName).Logger().WithTraceID = true

	clfg := conf.NewFromStringMap(map[string]any{
		"TimestampFormat": "2006-01-02 15:04:05",
		"Format":          "request,response",
	})
	gs, addr := testproto.NewPingGrpcService(t, grpc.ChainUnaryInterceptor(AccessLogger{}.UnaryServerInterceptor(clfg), applog()))
	assert.NotNil(t, gs)
	defer gs.Stop()

	conn, client := testproto.NewPingGrpcClient(t, context.Background(), addr)
	defer conn.Close()

	ctx := metadata.AppendToOutgoingContext(context.Background(), log.TraceIDKey, "some_trace_id")
	_, err := client.Ping(ctx, &testproto.PingRequest{
		Value: t.Name(),
	})
	assert.NoError(t, err)
	ls := logdata.String()
	assert.Contains(t, ls, "some_trace_id")
	assert.Contains(t, ls, "logger_test")
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
		want    func() *logtest.Buffer
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "info",
			args: args{
				cfg: conf.NewFromStringMap(map[string]any{}),
			},
			fields: fields{
				ctx: context.Background(),
				handler: func(ctx context.Context, req any) (any, error) {
					log.AppendToIncomingContext(ctx, zap.String("woocoo", "test"))
					return nil, nil
				},
				info: &grpc.UnaryServerInfo{FullMethod: "test"},
			},
			want: func() *logtest.Buffer {
				logdata := &logtest.Buffer{}
				log.Component(AccessLogComponentName).SetLogger(log.New(logtest.NewBuffLogger(logdata)))
				return logdata
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				logdata := i[0].(*logtest.Buffer)
				lines := logdata.Lines()
				assert.Len(t, lines, 1)
				assert.Contains(t, lines[0], "info")
				assert.Contains(t, lines[0], "woocoo")
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := tt.want()

			unaryServerInterceptor := AccessLogger{}.UnaryServerInterceptor(tt.args.cfg)
			got, err := unaryServerInterceptor(tt.fields.ctx, nil, tt.fields.info, tt.fields.handler)
			if !tt.wantErr(t, err, want) {
				return
			}
			assert.Nil(t, got)
		})
	}
}

func TestLoggerStreamServerInterceptor(t *testing.T) {
	interceptor := AccessLogger{}
	assert.Equal(t, "accessLog", interceptor.Name())
	logdata := &logtest.Buffer{}
	log.New(logtest.NewBuffLogger(logdata, zap.AddStacktrace(zap.ErrorLevel))).AsGlobal()
	UseContextLogger()
	t.Run("no panic", func(t *testing.T) {
		handlerCalled := false
		stream := &mockStream{}
		handler := func(srv any, stream grpc.ServerStream) error {
			// 2 log,this and access log
			logger.Ctx(stream.Context()).Info("test")
			handlerCalled = true
			return nil
		}
		err := interceptor.StreamServerInterceptor(conf.NewFromStringMap(map[string]any{
			"Format": "request,response",
		}))(nil, stream, &grpc.StreamServerInfo{FullMethod: "stream"}, handler)
		require.NoError(t, err)
		assert.True(t, handlerCalled)
		assert.Contains(t, logdata.String(), "info")
		assert.Contains(t, logdata.String(), "")
	})
}

func TestDefaultCodeToLevel(t *testing.T) {
	tests := []struct {
		code  codes.Code
		level zapcore.Level
	}{
		{codes.OK, zap.InfoLevel},
		{codes.Canceled, zap.InfoLevel},
		{codes.Unknown, zap.ErrorLevel},
		{codes.InvalidArgument, zap.InfoLevel},
		{codes.DeadlineExceeded, zap.WarnLevel},
		{codes.NotFound, zap.InfoLevel},
		{codes.AlreadyExists, zap.InfoLevel},
		{codes.PermissionDenied, zap.WarnLevel},
		{codes.Unauthenticated, zap.InfoLevel},
		{codes.ResourceExhausted, zap.WarnLevel},
		{codes.FailedPrecondition, zap.WarnLevel},
		{codes.Aborted, zap.WarnLevel},
		{codes.OutOfRange, zap.WarnLevel},
		{codes.Unimplemented, zap.ErrorLevel},
		{codes.Internal, zap.ErrorLevel},
		{codes.Unavailable, zap.WarnLevel},
		{codes.DataLoss, zap.ErrorLevel},
		{9999, zap.ErrorLevel}, // Unknown code
	}

	for _, tt := range tests {
		t.Run(tt.code.String(), func(t *testing.T) {
			level := DefaultCodeToLevel(tt.code)
			assert.Equal(t, tt.level, level)
		})
	}
}
