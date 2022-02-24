package logger

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"go.uber.org/zap"
	"time"
)

const (
	InnerPath = "innerPath"
)

var LogType = zap.String("type", "accessLog")

type Handler struct {
	exclude []string
	logger  *log.Logger
}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	h.exclude = append(h.exclude, cfg.StringSlice("exclude")...)
	h.logger = log.Global()
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
		for _, ex := range h.exclude {
			if spath == ex {
				shouldLog = false
				break
			}
		}
		if !shouldLog && len(c.Errors) == 0 {
			return
		}
		if raw != "" {
			path = path + "?" + raw
		}
		latency := time.Since(start)
		h.logger.Info(c.Request.URL.Path,
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

func (h *Handler) Shutdown() {
}
