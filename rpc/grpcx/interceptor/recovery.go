package interceptor

import (
	"context"
	"github.com/tsingsun/woocoo/pkg/conf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
	"runtime"
)

type RecoveryOptions struct {
	// Size of the stack to be printed.
	// Optional. Default value 4KB.
	StackSize int `json:"stackSize" yaml:"stackSize"`

	// DisableStackAll disables formatting stack traces of all other goroutines
	// into buffer after the trace for the current goroutine.
	// Optional. Default value false.
	DisableStackAll bool `json:"disableStackAll" yaml:"disableStackAll"`

	// DisablePrintStack disables printing stack trace.
	// Optional. Default value as false.
	DisablePrintStack bool `json:"disablePrintStack" yaml:"disablePrintStack"`
}

var (
	defaultRecoveryOptions = RecoveryOptions{
		StackSize:         4 << 10, // 4 KB
		DisableStackAll:   false,
		DisablePrintStack: false,
	}
)

// RecoveryUnaryServerInterceptor catches panics in processing unary requests and recovers.
func RecoveryUnaryServerInterceptor(cfg *conf.Configuration) grpc.UnaryServerInterceptor {
	if err := cfg.Unmarshal(&defaultRecoveryOptions); err != nil {
		panic(err)
	}
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		panicked := true
		defer func() {
			if r := recover(); r != nil || panicked {
				err = handleRecoverError(defaultRecoveryOptions, ctx, r)
			}
		}()
		resp, err = handler(ctx, req)
		panicked = false
		return resp, err
	}
}

// RecoveryStreamServerInterceptor returns a new streaming server interceptor for panic recovery.
func RecoveryStreamServerInterceptor(cfg *conf.Configuration) grpc.StreamServerInterceptor {
	if err := cfg.Unmarshal(&defaultRecoveryOptions); err != nil {
		panic(err)
	}

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		panicked := true
		defer func() {
			if r := recover(); r != nil || panicked {
				err = handleRecoverError(defaultRecoveryOptions, stream.Context(), r)
			}
		}()
		err = handler(srv, stream)
		panicked = false
		return err
	}
}

func handleRecoverError(ro RecoveryOptions, ctx context.Context, p interface{}) error {
	var stack []byte
	var length int
	if !ro.DisablePrintStack {
		stack = make([]byte, ro.StackSize)
		length = runtime.Stack(stack, !ro.DisableStackAll)
		stack = stack[:length]
	}

	grpclog.Errorf("%+v %s...", p, stack)
	return status.Errorf(codes.Internal, "panic: %v", p)
}
