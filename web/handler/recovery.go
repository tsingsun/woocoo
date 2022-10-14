package handler

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"go.uber.org/zap"
	"net/http"
	"net/http/httputil"
)

var (
	ErrRecovery = errors.New("internal server error")
)

// RecoveryMiddleware is a middleware which recovers from panics anywhere in the chain
// and handles the control to the centralized HTTPErrorHandler.
type RecoveryMiddleware struct {
}

func Recovery() *RecoveryMiddleware {
	return &RecoveryMiddleware{}
}

func (h *RecoveryMiddleware) Name() string {
	return "recovery"
}

func (h *RecoveryMiddleware) ApplyFunc(_ *conf.Configuration) gin.HandlerFunc {
	return gin.CustomRecovery(HandleRecoverError)
}

func (h *RecoveryMiddleware) Shutdown() {
}

func HandleRecoverError(c *gin.Context, err any) {
	httpRequest, _ := httputil.DumpRequest(c.Request, false)
	var ce *gin.Error
	if e, ok := err.(error); ok {
		ce = c.AbortWithError(http.StatusInternalServerError, e)
		ce.Type = gin.ErrorTypePrivate
	} else {
		_ = c.AbortWithError(http.StatusInternalServerError, ErrRecovery)
	}
	fc := GetLogCarrierFromGinContext(c)
	if fc != nil {
		fc.Fields = append(fc.Fields,
			zap.Any("panic", err),
			zap.String("request", string(httpRequest)),
			zap.Stack("stacktrace"),
		)
		return
	}
	if logger.Logger().DisableStacktrace {
		logger.Ctx(c).Error("[Recovery from panic]",
			zap.Any("error", err),
			zap.String("request", string(httpRequest)),
			zap.Stack("stacktrace"),
		)
	} else {
		logger.Ctx(c).Error("[Recovery from panic]",
			zap.Any("error", err),
			zap.String("request", string(httpRequest)),
		)
	}
}
