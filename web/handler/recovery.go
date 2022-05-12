package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"go.uber.org/zap"
	"net/http/httputil"
	"runtime"
)

type RecoveryConfig struct {
	// Size of the stack to be printed.
	// Optional. Default value 4KB.
	StackSize int `json:"stackSize" yaml:"stackSize"`

	// DisableStackAll disables formatting stack traces of all other goroutines
	// into buffer after the trace for the current goroutine.
	// Optional. Default value false.
	DisableStackAll bool `json:"disableStackAll" yaml:"disableStackAll"`

	// DisablePrintStack disables printing stack trace.
	// Optional. Default value as false.
	DisablePrintStack bool `json:"disablePrintStack" yaml:"disablePrintStack"`
}

var (
	defaultRecoveryConfig = RecoveryConfig{
		StackSize:         4 << 10, // 4 KB
		DisableStackAll:   false,
		DisablePrintStack: false,
	}
)

type RecoveryMiddleware struct {
}

func Recovery() *RecoveryMiddleware {
	return &RecoveryMiddleware{}
}

func (h *RecoveryMiddleware) Name() string {
	return "recovery"
}

func (h *RecoveryMiddleware) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	if err := cfg.Unmarshal(&defaultRecoveryConfig); err != nil {
		panic(err)
	}
	return gin.CustomRecovery(HandleRecoverError)
}

func (h RecoveryMiddleware) Shutdown() {
}

func HandleRecoverError(c *gin.Context, err interface{}) {
	var stack []byte
	var length int
	httpRequest, _ := httputil.DumpRequest(c.Request, false)
	if !defaultRecoveryConfig.DisablePrintStack {
		stack = make([]byte, defaultRecoveryConfig.StackSize)
		length = runtime.Stack(stack, !defaultRecoveryConfig.DisableStackAll)
		stack = stack[:length]
	}
	logger.Error("[Recovery from panic]",
		zap.Any("error", err),
		zap.String("request", string(httpRequest)),
		zap.ByteString("stack", stack),
	)
}
