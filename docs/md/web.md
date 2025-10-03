---
id: gin
---

# Gin

基于Gin组件的强大能力,WooCoo将Gin做为Web应用的引擎, 构建web应用也是非常简单的.

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
            # panic处理中间件    
            - recovery:
            # 访问日志中间件
            - accessLog:
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

由web的配置节点中,在`middlawares`中配置中间件,我们开始介绍内置的中间件.具体的代码可查看[handler](https://github.com/tsingsun/woocoo/tree/main/web/handler)

中间件的加载是有顺序的,因此定制程序配置文件时,请根据实际情况配置. 一般前置的几个中间件如下:

```yaml
recovery:  #...
accessLog: #...
errorHandle: #...
```

需要特别说明的是设置在`default`路由组节点下的中间件是全局的,将会应用到所有的路由中, 由于是顺序加载,因此`default`务必放在`routerGroups`最前面.
如果子路由组中配置了同名的中间件,两个是间件将同时生效,而不是覆盖.

## AccessLog

访问日志借助了高性能的log组件实现HTTP请求日志. 可定义输出格式.

使用方式,在配置中放入:
```yaml
middlewares:
  - accessLog: # 不设置任何使用默认配置      
```

可以自定义格式,支持的tag如下:

- id (Request ID or trace ID)
- remoteIp
- uri
- host
- method
- path
- protocol
- referer
- userAgent
- status
- error
- latency (纳秒)
- latencyHuman (可读性)
- bodyIn (请求体)
- bytesIn (进站大小)
- bytesOut (出站大小)
- header:NAME 
- query:NAME
- form:NAME
- context:NAME

```yaml
accessLog:
  format: "id,header:accept,context:tenantID,query:id"
```

log组件使用的全局的日志核心,因此错误等级与全局的错误等级是一致的.同时也提供了自定义等级,但该等级只能高于全局设置.请参考[logger](./logger.md)
```yaml
level: info #错误等级文本 
```

## Recovery

Recovery是用于程序从panic中恢复的recover,其结合日志及错误的中间件, 可在日志中输出错误的stack.用法也比较简单.直接在配置中放入.

```yaml
recovery: ## 目前为空配置.
```

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

我们也提供了一个方法来支持错误代码(code)和错误信息(message)的输出.
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

### 自定义错误映射

内置了Error的映射方法,你可以通过程序化来自定义错误映射,来应不同的需求, 有两类错误映射:

- codeMap: key:int,value:string ,你可以将http.StatusCode配置为错误码来映射错误信息.
- errorMap: key:error,value:string ,你可以将error配置为错误码来映射错误信息.适应于底层返回错误难以被识别的错误.

```go
customCodeMap:= map[int]any{
	// init by yourself
	500: "系统错误,请联系管理员",
	10000: "自定义错误信息",
}
customErrorMap:= map[int]any{
	"error pkg error": "系统错误,请联系管理员"
}
handler.SetErrorMap(customCodeMap, customErrorMap)

```

### 自定义错误解析

可以通过程序化来自定义处理程序,来应不同的需求:

```go
// ExampleErrorHandleMiddleware_customErrorParser is the example for customer ErrorHandle
func ExampleErrorHandleMiddleware_customErrorParser() {
	hdl := handler.ErrorHandle(handler.WithMiddlewareConfig(func(config any) {
		cfg := config.(*handler.ErrorHandleConfig)
		codeMap := map[uint64]any{
			10000: "miss required param",
			10001: "invalid param",
		}
		errorMap := map[interface{ Error() string }]string{
			http.ErrBodyNotAllowed: "username/password not correct",
		}
        cfg.Accepts = "application/json,application/xml"
        cfg.Message = "internal error"
        cfg.ErrorParser = func(c *gin.Context, public error) (int, any) {
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
	}))
	gin.Default().Use(hdl.ApplyFunc(nil))
}
```

### 模板化错误

在某些场景下,错误信息需要模板化,比如错误码,错误信息等需要多语言支持, 在这种情况下可以进阶协议.

前端可根据格式进行展示,如下RestApi:

```json
{
  "errors": [
    {
      "code": 10000,
      "message": "自定义错误信息: %s %f",
      "meta": ["string", 1.1]
    },
    {
      "code": 500,
      "message": "系统错误,请联系管理员.{{key}}",
      "meta": {
        "key": "value"
      }
    }
  ]
}
```

该协议在适配graphql时,会做如下变化:
```json
{
  "errors": [
    {
      "message": "自定义错误信息: %s %f", 
      "extensions": {
        "code": 10000,
        "meta": ["string", 1.1]
      }
    },
    {
      "message": "系统错误,请联系管理员.{{key}}",
      "extensions": {
        "code": 500,
        "meta": {
          "key": "value"
        }
      }
    }
  ]
}
```

## JSON Web Token(JWT)

基于JWT的应用非常广泛,因此也内置了该中间件.`jwt`支持了较多的功能:

- 各种签名方式,HS,RS,PS等.
- 支持各种token传递方式,Authorization Header及Query等可自定义.
- 默认验证成功后注入用户信息至上下文,方便获取用户信息,同时提供logout处理器.
- 支持集中式缓存验证,如token存至redis中进行有效性验证.

```yaml
jwt:
  signingMethod: "HS256"
  signingKey: "secret"
```
更多的具有配置内容,可查看代码实现.

## CORS

CORS是跨域资源共享的简称,由于web服务的特殊性,因此我们提供了CORS中间件,可通过配置文件进行配置.

```yaml
cors:
  # 简易配置可设置为 allowOrigins: ['*']
  allowOrigins: ["http://localhost:8080","https://github.com"]  
  # 简易配置可设置为: ["*"]
  # 默认值为 Origin,Content-Type,Accept,Authorization
  allowHeaders: "Origin,Content-Type,Accept,Authorization"
  # 默认值为 (GET, POST, PUT, PATCH, DELETE, HEAD, and OPTIONS)
  allowMethods: true
  # 用于控制是否允许浏览器发送跨域请求时携带认证信息（如 cookies、HTTP 认证信息等）
  allowCredentials: true
  # 表示预检请求的结果可以被缓存时长。默认12小时
  maxAge: "1h"
  # 是否允许通配符 http://some-domain/*, https://api.* or http://some.*.subdomain.com
  allowWildcard: false
```

更多的额外的配置可参考[github](https://github.com/gin-contrib/cors)中的`cors.Config`,对应在配置文件中加入即可.

关于CORS的参考: 

[预检请求](https://developer.chrome.com/blog/private-network-access-preflight?hl=zh-cn)

## 签名HTTP

针对HTTP Request签名主要为了保证请求的安全,可以防止数据被篡改,防止重放攻击.提供两种签名机制.

在本中间件中,也提供了指向缓存配置,可以轻松的融合.

- 简易签名: 
  
  是通过时间戳,随机数,token因素签名.
  是微信JS签名相似的,也被多家所采用,通过合理的参数设置,是一种性价比较高的机制.简单又安全,同时也无需对Body进行哈希.前各端都可以轻松实现.

  例子:

  ```
  Authorization: XXX-HMAC-SHA256 timestamp=1414587457;nonce=Wm3WZYTPz0wzccnW;Signature=OJZA/jnroXMK/sg3VBiUCdE4angcf9p40SmSMlwyN88="
  ```

- 规范签名:

  根据 Http Signature规范对涉及到请求要素(header,资源,密钥)进行签名,是标准的安全做法.

### 配置

以上两种都可以通过配置文件进行配置. `sign`对应规范签名,`tokenSign`对应简易签名

以下为关键配置:

- `signedLookups`: 需要签名的元素信息,key为签名元素, value为签名元素的来源,可以是header,query,context等,直接不填来源会默认header.
- `authLookup`: 签名信息写入位置. 默认是 `header:Authorization`,则会将签名信息写入到Authorization头中.
- `authScheme`: 当需要写入`header:Authorization` 一般要指定scheme;根据规范,一般如`XXX-HMAC-SHA1`等标准服务商.
- `authHeaders`: 需要签名写入到Header的元素.固定写入头的为`Signature`不需要配置,其他的都是可选的.
- `interval`: 在当前时间内的时间戳有效期,默认为5分钟.如果时间戳与当前时间相差超过该值,则认为是无效的签名.
- `ttl`: 签名缓存有效期,默认为24小时.配置该值时,请结合accessToken的有效期进行配置.至少应该是要大于accessToken的有效期. 
这样可以保证accessToken失效后,签名也失效.即例缓存被清除了,也可保证签名失效.

```yaml
## 简易签名
tokenSign:  
  signerConfig:
    authLookup: "header:Authorization"
    authScheme: "XXX-HMAC-SHA1"     
    authHeaders: ["timestamp","noncestr"]
    signedLookups:
      accessToken: header:Authorization>Bearer
      timestamp:
      noncestr:
      url: CanonicalUri  
  storeKey: signature
 ```

```yaml
##
sign:  
  signerConfig:
    authLookup: "header:Authorization"
    authScheme: "TEST-HMAC-SHA1" 
    authHeaderDelimiter: ";"
    signedLookups:
      - x-timestamp: "header"
      - content-type: "header"
      - content-length: ""
      - x-tenant-id: "header" 
    timestampKey: x-timestamp 
  storeKey: signature
```

签名算法中的值除了来源来header,query,context外,我们还约定了常见的值, 如:

- `CanonicalUri`:规范化Uri,到Path信息,但TokenSigner为去除了fragment的完整路径.
