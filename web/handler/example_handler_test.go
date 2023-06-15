package handler_test

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/web/handler"
)

// ExampleErrorHandleMiddleware_customErrorParser is the example for customer ErrorHandle
func ExampleErrorHandleMiddleware_customErrorParser() {
	hdl := handler.ErrorHandle(handler.WithMiddlewareConfig(func() any {
		codeMap := map[uint64]any{
			10000: "miss required param",
			10001: "invalid param",
		}
		errorMap := map[interface{ Error() string }]string{
			http.ErrBodyNotAllowed: "username/password not correct",
		}
		return &handler.ErrorHandleConfig{
			Accepts: "application/json,application/xml",
			Message: "internal error",
			ErrorParser: func(c *gin.Context, public error) (int, any) {
				var errs = make([]gin.H, len(c.Errors))
				for i, e := range c.Errors {
					if txt, ok := codeMap[uint64(e.Type)]; ok {
						errs[i] = gin.H{"code": i, "message": txt}
						continue
					}
					if txt, ok := errorMap[e.Err]; ok {
						errs[i] = gin.H{"code": i, "message": txt}
						continue
					}
					errs[i] = gin.H{"code": i, "message": e.Error()}
				}
				return 0, errs
			},
		}
	}))
	// use in web server
	gin.Default().Use(hdl.ApplyFunc(nil))
}
