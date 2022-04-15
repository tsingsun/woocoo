package logger

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"go.uber.org/zap"
	"io/ioutil"
	"time"
)

const (
	InnerPath = "innerPath"
)

var (
	LogType = zap.String("type", "accessLog")
	logger  = log.Component("webAccess")
)

type Handler struct {
	exclude     []string
	requestBody bool // whether log request body
}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	h.exclude = append(h.exclude, cfg.StringSlice("exclude")...)
	h.requestBody = cfg.Bool("requestBody")

	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		//body
		var bodyBytes []byte
		if h.requestBody {
			if c.Request.Body != nil {
				bodyBytes, _ = ioutil.ReadAll(c.Request.Body)
			}
			c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		}
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
		fields := []zap.Field{
			LogType,
			zap.String("clientIp", c.ClientIP()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("userAgent", c.Request.UserAgent()),
			zap.Int("bodySize", c.Writer.Size()),
			zap.String("error", c.Errors.ByType(gin.ErrorTypePrivate).String()),
		}
		if h.requestBody {
			fields = append(fields, zap.ByteString("body", bodyBytes))
		}
		logger.Info(c.Request.URL.Path,
			fields...,
		)
	}
}

func (h *Handler) Shutdown() {
}
