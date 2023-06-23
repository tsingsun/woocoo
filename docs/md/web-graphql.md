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
# 是否跳过该配置,用于启用或禁用该graphql配置
skip: false
# 是否启用鉴权组件,可详见鉴权组件说明
withAuthorization: false
# 鉴权组件所需要的应用名称.
appCode: ""
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
        
```
加载该中间件处理器.
```go
webSrv := web.New(
  web.RegisterMiddleware(gql.New())
)
// 将自行实现的graphql.ExecutableSchema关联到服务中,可多个
_, err := RegisterSchema(srv, &gqlSchema, &gqlSchema)
```

