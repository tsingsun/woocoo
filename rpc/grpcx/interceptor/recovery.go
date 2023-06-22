package interceptor

import (
	"context"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	defaultRecoveryOptions = RecoveryOptions{}
)

type (
	RecoveryOptions struct {
	}

	Recovery struct {
	}
)

// Name return the name of recovery interceptor
func (Recovery) Name() string {
	return "recovery"
}

// UnaryServerInterceptor catches panics in processing unary requests and recovers.
func (Recovery) UnaryServerInterceptor(cnf *conf.Configuration) grpc.UnaryServerInterceptor {
	if err := cnf.Unmarshal(&defaultRecoveryOptions); err != nil {
		panic(err)
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		panicked := true
		defer func() {
			if r := recover(); r != nil || panicked {
				err = handleRecoverError(defaultRecoveryOptions, ctx, r, log.CallerSkip+2)
			}
		}()
		resp, err = handler(ctx, req)
		panicked = false
		return resp, err
	}
}

// StreamServerInterceptor returns a new streaming server interceptor for panic recovery.
func (Recovery) StreamServerInterceptor(cnf *conf.Configuration) grpc.StreamServerInterceptor {
	if err := cnf.Unmarshal(&defaultRecoveryOptions); err != nil {
		panic(err)
	}
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		panicked := true
		defer func() {
			if r := recover(); r != nil || panicked {
				err = handleRecoverError(defaultRecoveryOptions, stream.Context(), r, log.CallerSkip+2)
			}
		}()
		err = handler(srv, stream)
		panicked = false
		return err
	}
}

// HandleRecoverError return recovery error when grpc occur panic. use the options of recovery interceptor.
//
// the method can use easy in biz code, like :
//
//	wg.Go(func() (err error) {
//		defer func() {
//			if r := recover(); r != nil {
//				err = interceptor.HandleRecoverError(ctx, r)
//			}
//		}()
func HandleRecoverError(ctx context.Context, p any) error {
	return handleRecoverError(defaultRecoveryOptions, ctx, p, 2)
}

// if use logger,let it log the stack trace
func handleRecoverError(_ RecoveryOptions, ctx context.Context, p any, stackSkip int) (err error) {
	err, ok := p.(error)
	if !ok {
		err = status.Errorf(codes.Internal, "%v", p)
	}
	if carrier, ok := log.FromIncomingContext(ctx); ok {
		carrier.Fields = append(carrier.Fields,
			zap.NamedError("panic", err),
			zap.StackSkip(log.StacktraceKey, stackSkip),
		)
		return err
	}
	logger.Ctx(ctx).WithOptions(zap.AddCallerSkip(stackSkip)).Error("[Recovery from panic]",
		zap.Error(err))

	return err
}
