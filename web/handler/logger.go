package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"go.uber.org/zap"
	"strings"
	"time"
)

const (
	InnerPath = "woocoo.innerPath"
)

var (
	defaultLoggerFormat = "id,remoteIp,host,method,uri,userAgent,status,error," +
		"latency,bytesIn,bytesOut"
)

type LoggerConfig struct {
	Skipper Skipper
	Exclude []string `json:"exclude" yaml:"exclude"`
	// Tags to construct the logger format.
	//
	// - id (Request ID)
	// - remoteIp
	// - uri
	// - host
	// - method
	// - path
	// - protocol
	// - referer
	// - userAgent
	// - status
	// - error
	// - latency (In nanoseconds)
	// - latencyHuman (Human readable)
	// - bytesIn (Bytes received)
	// - bytesOut (Bytes sent)
	// - header:<NAME>
	// - query:<NAME>
	// - form:<NAME>
	// - context:<NAME>
	//
	//
	// Optional. Default value DefaultLoggerConfig.Format.
	Format string `json:"format" yaml:"format"`
	tags   []string
}

type LoggerMiddleware struct{}

// Logger a new LoggerMiddleware,it is for handler registry
func Logger() *LoggerMiddleware {
	return &LoggerMiddleware{}
}

func (h *LoggerMiddleware) Name() string {
	return "accessLog"
}

// ApplyFunc build a gin.HandlerFunc for AccessLog middleware
func (h *LoggerMiddleware) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	opts := LoggerConfig{
		Format: defaultLoggerFormat,
	}
	if err := cfg.Unmarshal(&opts); err != nil {
		panic(err)
	}
	opts.tags = strings.Split(opts.Format, ",")
	if opts.Skipper == nil && len(opts.Exclude) > 0 {
		opts.Skipper = func(c *gin.Context) bool {
			// if has error,should log
			if len(c.Errors) > 0 {
				return false
			}
			path := c.Request.URL.Path
			for _, p := range opts.Exclude {
				if path == p {
					return true
				}
			}
			return false
		}
	} else {
		opts.Skipper = DefaultSkipper
	}
	return func(c *gin.Context) {
		start := time.Now()
		// Process request first
		c.Next()
		// c.Next() may change implicit skipper,so call it after c.Next()
		if opts.Skipper(c) {
			return
		}
		req := c.Request
		stop := time.Now()
		res := c.Writer
		latency := stop.Sub(start)
		var fields []zap.Field
		for _, tag := range opts.tags {
			switch tag {
			case "id":
				id := req.Header.Get("X-Request-Id")
				fields = append(fields, zap.String("id", id))
			case "remoteIp":
				fields = append(fields, zap.String("remoteIp", c.ClientIP()))
			case "host":
				fields = append(fields, zap.String("host", req.Host))
			case "uri":
				fields = append(fields, zap.String("uri", req.RequestURI))
			case "method":
				fields = append(fields, zap.String("method", req.Method))
			case "path":
				fields = append(fields, zap.String("path", c.GetString(InnerPath)))
			case "protocol":
				fields = append(fields, zap.String("protocol", req.Proto))
			case "referer":
				fields = append(fields, zap.String("referer", req.Referer()))
			case "userAgent":
				fields = append(fields, zap.String("userAgent", req.UserAgent()))
			case "status":
				fields = append(fields, zap.Int("status", res.Status()))
			case "error":
				if len(c.Errors) > 0 {
					fields = append(fields, zap.String("error", c.Errors.ByType(gin.ErrorTypePrivate).String()))
				}
			case "latency":
				fields = append(fields, zap.Duration("latency", latency))
			case "latencyHuman":
				fields = append(fields, zap.String("latencyHuman", latency.String()))
			case "bytesIn":
				fields = append(fields, zap.Int64("bytesIn", req.ContentLength))
			case "bytesOut":
				fields = append(fields, zap.Int("bytesOut", res.Size()))
			default:
				switch {
				case strings.HasPrefix(tag, "header:"):
					fields = append(fields, zap.String(tag, c.Request.Header.Get(tag[7:])))
				case strings.HasPrefix(tag, "query:"):
					fields = append(fields, zap.String(tag, c.Query(tag[6:])))
				case strings.HasPrefix(tag, "form:"):
					fields = append(fields, zap.String(tag, c.PostForm(tag[5:])))
				case strings.HasPrefix(tag, "cookie:"):
					cookie, err := c.Cookie(tag[7:])
					if err == nil {
						fields = append(fields, zap.String(tag, cookie))
					}
				case strings.HasPrefix(tag, "context:"):
					val, ok := c.Get(tag[8:])
					if ok {
						fields = append(fields, zap.Any(tag, val))
					}
				}
			}
		}
		logger.Info("", fields...)
	}
}

// Shutdown does nothing for logger
func (h *LoggerMiddleware) Shutdown() {
}
