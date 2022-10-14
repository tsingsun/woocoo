package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"strings"
	"time"
)

type loggerTagType int

const (
	loggerTagTypeString loggerTagType = iota
	loggerTagTypeHeader
	loggerTagTypeQuery
	loggerTagTypeForm
	loggerTagTypeCookie
	loggerTagTypeContext
)

var (
	accessLoggerName    = "web.accessLog"
	defaultLoggerFormat = "id,remoteIp,host,method,uri,userAgent,status,error," +
		"latency,bytesIn,bytesOut"
)

type LoggerConfig struct {
	Skipper Skipper
	Exclude []string `json:"exclude" yaml:"exclude"`
	// Tags to construct the logger format.
	//
	// - id (Request ID or trace ID)
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
	tags   []loggerTag
}

type loggerTag struct {
	v string
	t loggerTagType
	k string
}

type LoggerMiddleware struct {
	logger log.ComponentLogger
}

// AccessLog a new LoggerMiddleware,it is for handler registry
func AccessLog() *LoggerMiddleware {
	al := &LoggerMiddleware{}
	operator := logger.Logger().WithOptions(zap.AddStacktrace(zapcore.FatalLevel + 1))
	logger := log.Component(accessLoggerName)
	logger.SetLogger(operator)
	al.logger = logger
	return al
}

func (h *LoggerMiddleware) Name() string {
	return "accessLog"
}

func (h *LoggerMiddleware) buildTag(format string) (tags []loggerTag) {
	ts := strings.Split(format, ",")
	for _, tag := range ts {
		switch {
		case strings.HasPrefix(tag, "header:"):
			tags = append(tags, loggerTag{v: tag, t: loggerTagTypeHeader, k: tag[7:]})
		case strings.HasPrefix(tag, "query:"):
			tags = append(tags, loggerTag{v: tag, t: loggerTagTypeQuery, k: tag[6:]})
		case strings.HasPrefix(tag, "form:"):
			tags = append(tags, loggerTag{v: tag, t: loggerTagTypeForm, k: tag[5:]})
		case strings.HasPrefix(tag, "cookie:"):
			tags = append(tags, loggerTag{v: tag, t: loggerTagTypeCookie, k: tag[7:]})
		case strings.HasPrefix(tag, "context:"):
			tags = append(tags, loggerTag{v: tag, t: loggerTagTypeContext, k: tag[8:]})
		default:
			tags = append(tags, loggerTag{v: tag, t: loggerTagTypeString})
		}
	}
	return
}

// ApplyFunc build a gin.HandlerFunc for AccessLog middleware
func (h *LoggerMiddleware) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	opts := LoggerConfig{
		Format: defaultLoggerFormat,
	}
	if err := cfg.Unmarshal(&opts); err != nil {
		panic(err)
	}
	opts.tags = h.buildTag(opts.Format)
	if opts.Skipper == nil && len(opts.Exclude) > 0 {
		opts.Skipper = func(c *gin.Context) bool {
			// if it has error,should log
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
		logCarrier := log.NewCarrier()
		c.Set(accessLoggerName, logCarrier)
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
		path := c.Request.URL.Path

		fields := make([]zap.Field, len(opts.tags))
		privateErr := false
		for i, tag := range opts.tags {
			switch tag.v {
			case "id":
				id := req.Header.Get("X-Request-Id")
				fields[i] = zap.String(log.TraceID, id)
			case "remoteIp":
				fields[i] = zap.String("remoteIp", c.ClientIP())
			case "host":
				fields[i] = zap.String("host", req.Host)
			case "uri":
				fields[i] = zap.String("uri", req.RequestURI)
			case "method":
				fields[i] = zap.String("method", req.Method)
			case "path":
				if c.Request.URL.RawQuery != "" {
					path = path + "?" + c.Request.URL.RawQuery
				}
				fields[i] = zap.String("path", path)
			case "protocol":
				fields[i] = zap.String("protocol", req.Proto)
			case "referer":
				fields[i] = zap.String("referer", req.Referer())
			case "userAgent":
				fields[i] = zap.String("userAgent", req.UserAgent())
			case "status":
				fields[i] = zap.Int("status", res.Status())
			case "error":
				if len(c.Errors) > 0 {
					v := c.Errors.ByType(gin.ErrorTypePrivate).String()
					if v != "" {
						privateErr = true
						fields[i] = zap.String("error", c.Errors.ByType(gin.ErrorTypePrivate).String())
					}
				}
			case "latency":
				fields[i] = zap.Duration("latency", latency)
			case "latencyHuman":
				fields[i] = zap.String("latencyHuman", latency.String())
			case "bytesIn":
				fields[i] = zap.Int64("bytesIn", req.ContentLength)
			case "bytesOut":
				fields[i] = zap.Int("bytesOut", res.Size())
			default:
				switch tag.t {
				case loggerTagTypeHeader:
					fields[i] = zap.String(tag.v, c.Request.Header.Get(tag.k))
				case loggerTagTypeQuery:
					fields[i] = zap.String(tag.v, c.Query(tag.k))
				case loggerTagTypeForm:
					fields[i] = zap.String(tag.v, c.PostForm(tag.k))
				case loggerTagTypeCookie:
					cookie, err := c.Cookie(tag.k)
					if err == nil {
						fields[i] = zap.String(tag.v, cookie)
					}
				case loggerTagTypeContext:
					val, ok := c.Get(tag.k)
					if ok {
						fields[i] = zap.Any(tag.v, val)
					}
				}
			}
			if fields[i].Type == zapcore.UnknownType {
				fields[i] = zap.Skip()
			}
		}
		if fc := getLogCarrierFromGinContext(c); fc != nil && len(fc.Fields) > 0 {
			fields = append(fields, fc.Fields...)
		}
		clog := h.logger.Ctx(c)
		if privateErr {
			clog.Error("", fields...)
		} else {
			clog.Info("", fields...)
		}
		log.PutLoggerWithCtxToPool(clog)
	}
}

// Shutdown does nothing for logger
func (h *LoggerMiddleware) Shutdown() {
}

func getLogCarrierFromGinContext(c *gin.Context) *log.FieldCarrier {
	if fc, ok := c.Get(accessLoggerName); ok {
		return fc.(*log.FieldCarrier)
	}
	return nil
}
