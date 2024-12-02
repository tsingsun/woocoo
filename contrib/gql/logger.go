package gql

import (
	"context"
	"github.com/99designs/gqlgen/graphql"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/web/handler"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
)

// StreamLogger is a graphql middleware for log stream response.
//
// Use the web access logger to log stream response.
type StreamLogger struct {
	config handler.LoggerConfig
	logger log.ComponentLogger
}

func (s *StreamLogger) Name() string {
	return handler.AccessLogName
}

func newStreamLogger() *StreamLogger {
	s := &StreamLogger{
		logger: log.Component(handler.AccessLogComponentName),
		config: handler.LoggerConfig{
			Format: "host,remoteIp,latency,error,resp",
		},
	}
	return s
}

// ApplyFunc build graphql response middleware for log stream response.
func (s *StreamLogger) ApplyFunc(cfg *conf.Configuration) graphql.ResponseMiddleware {
	if err := cfg.Unmarshal(&s.config); err != nil {
		panic(err)
	}
	s.config.BuildTag(s.config.Format)

	return func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
		c, err := FromIncomingContext(ctx)
		if err != nil {
			return next(ctx)
		}
		if !isStreamConnection(c) {
			return next(ctx)
		}
		start := time.Now()
		resp := next(ctx)
		if resp == nil {
			return nil
		}
		latency := time.Now().Sub(start)
		fields := make([]zap.Field, len(s.config.Tags))
		for i, tag := range s.config.Tags {
			switch tag.FullKey {
			case "remoteIp":
				fields[i] = zap.String("remoteIp", c.ClientIP())
			case "latency":
				fields[i] = zap.Duration("latency", latency)
			case "latencyHuman":
				fields[i] = zap.String("latencyHuman", latency.String())
			case "error":
				if len(resp.Errors) > 0 {
					fields[i] = zap.String("error", resp.Errors.Error())
				}
			case "host":
				fields[i] = zap.String("host", c.Request.Host)
			case "resp":
				fields[i] = zap.Any("resp", resp.Data)
			}
			if fields[i].Type == zapcore.UnknownType {
				fields[i] = handler.LoggerFieldSkip
			}
		}
		if fc := handler.GetLogCarrierFromGinContext(c); fc != nil && len(fc.Fields) > 0 {
			fields = append(fields, fc.Fields...)
		}
		clog := s.logger.Ctx(c)
		if len(resp.Errors) != 0 {
			clog.Error("", fields...)
		} else {
			clog.Info("", fields...)
		}
		return resp
	}
}
