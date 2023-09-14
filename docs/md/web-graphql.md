---
id: graphql
---
# GraphQL

GraphQL是一种用于API的查询语言,它提供了一种更高效、强大和灵活的替代REST的方式.它由Facebook开发并开源,现在由GraphQL基金会维护.
在后台API开发中已经越来越流行.

在woocoo中可以很容易的集成GraphQL服务.并且内置了GraphiQL,可以方便的进行调试.
而服务端的Graph程序可以借助gqlgen等工具生成实现代码后集成到woocoo中.

```bash
go get github.com/tsingsun/woocoo/contrib/gql@main
```
配置如下:
```yaml
# 路由组名称,默认在根路由组中
group: "/graphql"
# 查询路径,默认为/query
queryPath: /query
# GraphiQL文档路径,默认为/
docPath: /
# GraphIQL查询地址,默认为group+queryPath,
endpoint: ""
# 是否启用鉴权组件,可详见鉴权组件说明
withAuthorization: false
# 鉴权组件所需要的应用名称.
appCode: ""
# iGraphql界面默认的http header. 可以默认设置一些鉴权参数用于免配置快速使用.
header: map[string]string
# gql中间件,有别于web中间件.分为两种. 可采用web handler做为gql中间件.
middlewares:
  # Operation中间件.
  operation:
  # Response中间件.  
  response:
```

在web配置中以中间件方式加入:
```yaml
web:
  server:
    addr: :8080
  engine:    
    routerGroups:
      - default:                    
          middlewares:
            - graphql:
                # 配置...
      - api:
        # .....
```
加载该中间件处理器.
```go
webSrv := web.New(
    gql.RegistryMiddleware()
)
// 将自行实现的graphql.ExecutableSchema关联到服务中,可多个
_, err := RegisterSchema(srv, &gqlSchema, &gqlSchema)
```

> 多个服务注册时,需要注意Schema参数的顺序,需要与配置中的顺序一致.

## gql中间件.

contrib/gql包实现了web middleware的转化为gqlgen中间件,主要配置和web中间件的配置是一致的.根据实际情况选择
```yaml
web:
  server:
    addr: :8080
  engine:    
    routerGroups:
      - default:                    
          middlewares:
            - accessLog:
            - graphql:
                middlewares:
                  operation:
                    - jwt: # ...
```