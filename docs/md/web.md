---
id: web
---

# Web

由于gin组件的强大能力,构建web应用也是非常简单的.而woocoo目前专注于服务端API开发.

```go
	// 取全局程序配置
	cnf := conf.Global().
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

