package handler

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"net/http/httputil"

	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
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
	return gin.CustomRecovery(func(c *gin.Context, err any) {
		HandleRecoverError(c, err, 4)
	})
}

func (h *RecoveryMiddleware) Shutdown(_ context.Context) error {
	return nil
}

func HandleRecoverError(c *gin.Context, p any, stackSkip int) {
	httpRequest, _ := httputil.DumpRequest(c.Request, false)
	err, ok := p.(error)
	// gin private error not show to user
	if ok {
		AbortWithError(c, http.StatusInternalServerError, err)
	} else {
		AbortWithError(c, http.StatusInternalServerError, ErrRecovery)
		err = fmt.Errorf("%v", p)
	}
	fc := GetLogCarrierFromGinContext(c)
	if fc != nil {
		fc.Fields = append(fc.Fields,
			zap.NamedError("panic", err),
			zap.String("request", string(httpRequest)),
			zap.StackSkip(log.StacktraceKey, stackSkip),
		)
		return
	}
	logger.Ctx(c).WithOptions(zap.AddCallerSkip(stackSkip)).Error("[Recovery from panic]",
		zap.Error(err),
		zap.String("request", string(httpRequest)),
	)
}
