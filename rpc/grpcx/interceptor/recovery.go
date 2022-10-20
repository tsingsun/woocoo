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

type RecoveryOptions struct {
}

var (
	defaultRecoveryOptions = RecoveryOptions{}
)

// RecoveryUnaryServerInterceptor catches panics in processing unary requests and recovers.
func RecoveryUnaryServerInterceptor(cnf *conf.Configuration) grpc.UnaryServerInterceptor {
	if err := cnf.Unmarshal(&defaultRecoveryOptions); err != nil {
		panic(err)
	}
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		panicked := true
		defer func() {
			if r := recover(); r != nil || panicked {
				err = handleRecoverError(defaultRecoveryOptions, ctx, r, 0)
			}
		}()
		resp, err = handler(ctx, req)
		panicked = false
		return resp, err
	}
}

// RecoveryStreamServerInterceptor returns a new streaming server interceptor for panic recovery.
func RecoveryStreamServerInterceptor(cnf *conf.Configuration) grpc.StreamServerInterceptor {
	if err := cnf.Unmarshal(&defaultRecoveryOptions); err != nil {
		panic(err)
	}

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		panicked := true
		defer func() {
			if r := recover(); r != nil || panicked {
				err = handleRecoverError(defaultRecoveryOptions, stream.Context(), r, 0)
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
func HandleRecoverError(ctx context.Context, p interface{}) error {
	return handleRecoverError(defaultRecoveryOptions, ctx, p, 1)
}

// if use logger,let it log the stack trace
func handleRecoverError(_ RecoveryOptions, ctx context.Context, p interface{}, stackSkip int) (err error) {
	err = status.Errorf(codes.Internal, "panic: %v", p)
	if carrier, ok := log.FromIncomingContext(ctx); ok {
		carrier.Fields = append(carrier.Fields, zap.StackSkip(log.StacktraceKey, 3+stackSkip))
		return err
	}
	if logger.Logger().DisableStacktrace {
		logger.Ctx(ctx).Error("[Recovery from panic]",
			zap.Any("error", p),
			zap.StackSkip(log.StacktraceKey, 3+stackSkip),
		)
	} else {
		logger.Logger().WithOptions(zap.AddCallerSkip(6+stackSkip)).Ctx(ctx).Error("[Recovery from panic]",
			zap.Any("error", p),
		)
	}
	return err
}
