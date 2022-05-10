package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"go.uber.org/zap"
	"strings"
	"time"
)

const (
	InnerPath = "innerPath"
)

var (
	defaultLoggerFormat = "id,remoteIp,host,method,uri,userAgent,status,error," +
		"latency,bytesIn,bytesOut"
)

type LoggerOptions struct {
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
	opts := LoggerOptions{
		Format: defaultLoggerFormat,
	}
	if err := cfg.Unmarshal(&opts); err != nil {
		panic(err)
	}
	opts.tags = strings.Split(opts.Format, ",")

	return func(c *gin.Context) {
		req := c.Request
		start := time.Now()
		// Process request
		c.Next()
		res := c.Writer
		shouldLog := true
		path := c.GetString(InnerPath)
		if path == "" {
			path = req.URL.Path
		}
		for _, ex := range opts.Exclude {
			if path == ex {
				shouldLog = false
				break
			}
		}
		if !shouldLog && len(c.Errors) == 0 {
			return
		}
		stop := time.Now()
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
				fields = append(fields, zap.String("path", path))
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
				}
			}
		}
		logger.Info("", fields...)
	}
}

// Shutdown does nothing for logger
func (h *LoggerMiddleware) Shutdown() {
}
