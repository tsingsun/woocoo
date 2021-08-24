package logger

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/web/handler"
	"go.uber.org/zap"
	"time"
)

const (
	InnerPath = "innerPath"
)

var LogType = zap.String("type", "accessLog")

type options struct {
	exclude []string
	logger  *log.Logger
}

var defaultOptions = &options{
	exclude: []string{},
	logger:  log.Global(),
}

func AccessLogHandler(logger *log.Logger) handler.HandlerApplyFunc {
	o := &options{}
	*o = *defaultOptions
	o.logger = logger
	return func(cfg *conf.Configuration) gin.HandlerFunc {
		o.exclude = append(o.exclude, cfg.StringSlice("exclude")...)
		return func(c *gin.Context) {
			// Start timer
			start := time.Now()
			path := c.Request.URL.Path
			raw := c.Request.URL.RawQuery
			// Process request
			c.Next()
			shouldLog := true
			spath, ok := c.Get(InnerPath)
			if !ok {
				spath = path
			}
			for _, ex := range o.exclude {
				if spath == ex {
					shouldLog = false
				}
			}
			if !shouldLog && c.Errors == nil {
				return
			}
			if raw != "" {
				path = path + "?" + raw
			}
			latency := time.Since(start)
			o.logger.Info(c.Request.URL.Path,
				LogType,
				zap.String("clientIp", c.ClientIP()),
				zap.String("method", c.Request.Method),
				zap.String("path", path),
				zap.Int("status", c.Writer.Status()),
				zap.Int64("latency", latency.Milliseconds()),
				zap.String("userAgent", c.Request.UserAgent()),
				zap.Int("bodySize", c.Writer.Size()),
				zap.String("error", c.Errors.ByType(gin.ErrorTypePrivate).String()),
			)
		}
	}
}
