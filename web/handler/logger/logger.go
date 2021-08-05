package logger

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/web/handler"
	"go.uber.org/zap"
	"time"
)

func AccessLogHandler(logger *log.Logger) handler.HandlerApplyFunc {
	return func(cfg *conf.Configuration) gin.HandlerFunc {
		return func(c *gin.Context) {
			// Start timer
			start := time.Now()
			path := c.Request.URL.Path
			raw := c.Request.URL.RawQuery
			// Process request
			c.Next()

			if raw != "" {
				path = path + "?" + raw
			}
			latency := time.Since(start)
			logger.Info(c.Request.URL.Path,
				zap.String("clientIp", c.ClientIP()),
				zap.String("method", c.Request.Method),
				zap.String("path", path),
				zap.Int("status", c.Writer.Status()),
				zap.Duration("latency", latency),
				zap.String("userAgent", c.Request.UserAgent()),
				zap.Int("bodySize", c.Writer.Size()),
				zap.String("error", c.Errors.ByType(gin.ErrorTypePrivate).String()),
			)
		}
	}
}
