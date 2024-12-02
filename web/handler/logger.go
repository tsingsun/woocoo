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
	AccessLogComponentName = "web.accessLog"
	defaultLoggerFormat    = "id,remoteIp,host,method,uri,userAgent,status,error," +
		"latency,bytesIn,bytesOut"
	LoggerFieldSkip = zap.Skip()
)

type loggerTag struct {
	FullKey string
	typ     loggerTagType
	key     string
}

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
	Format string        `json:"format" yaml:"format"`
	Level  zapcore.Level `json:"level" yaml:"level"`
	Tags   []loggerTag
}

func (h *LoggerConfig) BuildTag(format string) {
	var tags []loggerTag
	ts := strings.Split(format, ",")
	for _, tag := range ts {
		tag = strings.TrimSpace(tag)
		switch {
		case strings.HasPrefix(tag, "header:"):
			tags = append(tags, loggerTag{FullKey: tag, typ: loggerTagTypeHeader, key: tag[7:]})
		case strings.HasPrefix(tag, "query:"):
			tags = append(tags, loggerTag{FullKey: tag, typ: loggerTagTypeQuery, key: tag[6:]})
		case strings.HasPrefix(tag, "form:"):
			tags = append(tags, loggerTag{FullKey: tag, typ: loggerTagTypeForm, key: tag[5:]})
		case strings.HasPrefix(tag, "cookie:"):
			tags = append(tags, loggerTag{FullKey: tag, typ: loggerTagTypeCookie, key: tag[7:]})
		case strings.HasPrefix(tag, "context:"):
			tags = append(tags, loggerTag{FullKey: tag, typ: loggerTagTypeContext, key: tag[8:]})
		default:
			tags = append(tags, loggerTag{FullKey: tag, typ: loggerTagTypeString})
		}
	}
	h.Tags = tags
}

// LoggerMiddleware is a middleware that logs each request.
type LoggerMiddleware struct {
	config LoggerConfig
	logger log.ComponentLogger
}

// NewAccessLog a new LoggerMiddleware,it is for handler registry
func NewAccessLog() *LoggerMiddleware {
	return &LoggerMiddleware{
		config: LoggerConfig{
			Format: defaultLoggerFormat,
			Level:  zapcore.InvalidLevel,
		},
	}
}

// AccessLog is the access logger middleware apply function. see MiddlewareNewFunc
func AccessLog() Middleware {
	mw := NewAccessLog()
	return mw
}

func (h *LoggerMiddleware) Name() string {
	return AccessLogName
}

// zap.AddStacktrace(zapcore.FatalLevel + 1) force to not add stack.
// zap.WithCaller(false) accessLog no need to.
func (h *LoggerMiddleware) buildLogger() {
	opts := []zap.Option{
		zap.AddStacktrace(zapcore.FatalLevel + 1), zap.WithCaller(false),
	}
	if h.config.Level != zapcore.InvalidLevel {
		opts = append(opts, zap.IncreaseLevel(h.config.Level))
	}
	olog := logger.Logger(log.WithOriginalLogger())
	operator := olog.WithOptions(opts...)
	h.logger = log.Component(AccessLogComponentName)
	h.logger.SetLogger(operator)
}

// ApplyFunc build a gin.HandlerFunc for NewAccessLog middleware
func (h *LoggerMiddleware) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	if err := cfg.Unmarshal(&h.config); err != nil {
		panic(err)
	}
	h.config.BuildTag(h.config.Format)
	h.buildLogger()
	if h.config.Skipper == nil && len(h.config.Exclude) > 0 {
		pm := StringsToMap(h.config.Exclude)
		h.config.Skipper = func(c *gin.Context) bool {
			// if it has error,should log
			if len(c.Errors) > 0 {
				return false
			}
			return PathSkip(pm, c.Request.URL)
		}
	} else {
		h.config.Skipper = DefaultSkipper
	}
	traceIDKey := h.logger.Logger().TraceIDKey
	return func(c *gin.Context) {
		start := time.Now()
		logCarrier := log.NewCarrier()
		c.Set(AccessLogComponentName, logCarrier)
		c.Next()
		if h.config.Skipper(c) {
			return
		}
		req := c.Request
		res := c.Writer
		latency := time.Now().Sub(start)
		path := c.Request.URL.Path

		fields := make([]zap.Field, len(h.config.Tags))
		privateErr := false
		for i, tag := range h.config.Tags {
			switch tag.FullKey {
			case "id":
				id := req.Header.Get("X-Request-Id")
				fields[i] = zap.String(traceIDKey, id)
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
						fields[i] = zap.String("error", v)
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
				switch tag.typ {
				case loggerTagTypeHeader:
					fields[i] = zap.String(tag.FullKey, c.Request.Header.Get(tag.key))
				case loggerTagTypeQuery:
					fields[i] = zap.String(tag.FullKey, c.Query(tag.key))
				case loggerTagTypeForm:
					fields[i] = zap.String(tag.FullKey, c.PostForm(tag.key))
				case loggerTagTypeCookie:
					cookie, err := c.Cookie(tag.key)
					// if no found, skip
					if err == nil {
						fields[i] = zap.String(tag.FullKey, cookie)
					}
				case loggerTagTypeContext:
					val := c.Value(tag.key)
					if val != nil {
						fields[i] = zap.Any(tag.FullKey, val)
					}
				}
			}
			if fields[i].Type == zapcore.UnknownType {
				fields[i] = LoggerFieldSkip
			}
		}
		if fc := GetLogCarrierFromGinContext(c); fc != nil && len(fc.Fields) > 0 {
			fields = append(fields, fc.Fields...)
		}
		clog := h.logger.Ctx(c)
		if privateErr {
			clog.Error("", fields...)
		} else {
			clog.Info("", fields...)
		}
	}
}

func GetLogCarrierFromGinContext(c *gin.Context) *log.FieldCarrier {
	if fc, ok := c.Get(AccessLogComponentName); ok {
		return fc.(*log.FieldCarrier)
	}
	return nil
}
