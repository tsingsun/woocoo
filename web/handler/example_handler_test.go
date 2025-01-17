package handler_test

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web/handler"
	"net/http"
)

// ExampleErrorHandleMiddleware_customErrorParser is the example for customer ErrorHandle
func ExampleErrorHandleMiddleware_customErrorParser() {
	hdl := handler.NewErrorHandle(handler.WithMiddlewareConfig(func(config any) {
		codeMap := map[uint64]any{
			10000: "miss required param",
			10001: "invalid param",
		}
		errorMap := map[interface{ Error() string }]string{
			http.ErrBodyNotAllowed: "username/password not correct",
		}
		c := config.(*handler.ErrorHandleConfig)
		c.Accepts = "application/json,application/xml"
		c.Message = "internal error"
		c.ErrorParser = func(c *gin.Context, public error) (int, any) {
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
		}
	}))
	// use in web server, you can pass a custom conf.Configuration
	gin.Default().Use(hdl.ApplyFunc(conf.New()))
}
