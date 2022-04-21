package recovery

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"go.uber.org/zap"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

const (
	RecoveryHandlerName = "recovery"
)

type Handler struct {
	stack bool
}

func New() *Handler {
	return &Handler{stack: true}
}

func (h *Handler) Name() string {
	return RecoveryHandlerName
}

func (h *Handler) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				HandleRecoverError(c, err, log.Global(), h.stack)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}

func (h Handler) Shutdown() {
}

func HandleRecoverError(c *gin.Context, err interface{}, logger log.ComponentLogger, stack bool) {
	// Check for a broken connection, as it is not really a
	// condition that warrants a panic stack trace.
	var brokenPipe bool
	if ne, ok := err.(*net.OpError); ok {
		if se, ok := ne.Err.(*os.SyscallError); ok {
			if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
				brokenPipe = true
			}
		}
	}

	httpRequest, _ := httputil.DumpRequest(c.Request, false)
	if brokenPipe {
		logger.Error(c.Request.URL.Path,
			zap.Any("error", err),
			zap.String("request", string(httpRequest)),
		)
		// If the connection is dead, we can't write a status to it.
		c.Error(err.(error)) // nolint: errcheck
		c.Abort()
		return
	}

	if stack {
		logger.Error("[Recovery from panic]",
			zap.Time("time", time.Now()),
			zap.Any("error", err),
			zap.String("request", string(httpRequest)),
			zap.String("stack", string(debug.Stack())),
		)
	} else {
		logger.Error("[Recovery from panic]",
			zap.Time("time", time.Now()),
			zap.Any("error", err),
			zap.String("request", string(httpRequest)),
		)
	}
}
