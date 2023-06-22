---
id: web
---

# Web

由于gin组件的强大能力,构建web应用也是非常简单的.而woocoo目前专注于服务端API开发.

```go
	// 取全局程序配置
	cnf := conf.Global()
	// 传入web节点配置
	webSrv := web.New(web.WithConfiguration(cnf.Sub("web")))
	// 采用gin原生的写法
	webSrv.Router().GET("/", func(c *gin.Context) {
		c.String(200, "hello world")
	})
```

首先我们来看一下配置:
```yaml
# root节点,可自己指定
web:
  # 服务器节点
  server:
    # 服务地址,符合地址要求的字符串
    addr: 0.0.0.0:10001
    # 如果启动tls时
    tls: 
      cert: ""
      key: ""
  # 服务引擎配置,子级等同于gin的router    
  engine:
    redirectTrailingSlash: false
    remoteIPHeaders:
      - X-Forwarded-For
      - X-Real-XIP
    # 路由组
    routerGroups:
      # 路由组名. default是默认路径为"/"的默认路由
      - default:
          # 路由组路径,非default节点必须配置
          basePath: "/"
          # 中间件配置
          middlewares:
            # 访问日志中间件
            - accessLog:
            # 异常处理中间件    
            - recovery:
            # 错误处理中间件    
            - errorHandle:
```

## engine

`engine`配置节实质是指向了`gin.Engine`结构体,通过将结构体的公共字段自行引入配置即可初始化常规的服务配置.

## 路由

路由是web服务中最重要的组件,可配置的路由同样是基于Gin.

由于路由的写法仍然使用gin的,与配置文件配合使用:

```go
    // 配置文件:
    // group:
    //   basePath: "/group"
    //   middlewares: .....
	webSrv.Router().GET("/group", func(c *gin.Context) {
		c.String(200, "hello world")
	})

```

上面的写法会使用配置文件中的设置,不需要在程序中显式引入分组及中间件.

# 中间件

由web的配置节点中,在`middlawares`中配置中间件,我们开始介绍内置的中间件.

## 错误处理

我们提倡通过从中间件和处理程序返回错误来集中处理HTTP错误,统一将错误直接或转换后输入到客户端.

内置的ErrorHandle提供了常规的错误处理机制.可通过配置文件配置.

```yaml
errorHandle:
  accepts: "application/json,application/xml" # http accepts接受的数据类型
  message: "系统错误,请联系管理员" # 针对私有错误错误信息
```

首先我们借用了Gin的错误定义,将错误区分为Public和Private两种类型. Public错误认为是可公开的,可以直接输出到客户端,
Private错误将会被记录到日志中,同时以配置文件中的message输出到客户端.对于这类型的Error,Code默认为Http Status.

默认通过gin.Context方法`c.Error(error)`方法产生的为private类型的错误.

我们也提供了一个方法来支持错误代表(code)和错误信息(message)的输出.
```go
// SetContextError set the error to Context,and the error will be handled by ErrorHandleMiddleware
func SetContextError(c *gin.Context, code int, err error) {
	ce := c.Error(err)
	ce.Type = gin.ErrorType(code)
}
```

最终error的输出格式类似如下:
```
{
    "errors": [
        {
            "code": 10000,
            "message": "自定义错误信息"
        },
        {
            "code": 500,
            "message": "系统错误,请联系管理员"
        }
    ]
}
```

同时也可以通过程序化来自定义处理程序,来应不同的需求:

```go
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
	gin.Default().Use(hdl.ApplyFunc(nil))
}
```