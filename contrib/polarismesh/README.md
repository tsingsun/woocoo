# polarismesh

this project was inspired by [grpc-go-polaris](github.com/polarismesh/grpc-go-polaris).

目前支持的北极星服务治理功能如下:

- [x] 服务注册及发现
- [x] 动态路由
- [] 服务熔断
- [x] 访问限流:部分能力
  - 根据请求方法进行限流.
- [] 配置中心:

## Client

使用普通的Grpc Client, 格式: `polaris://{namespace}/{service}?[options={query}]`作为请求目标:

query: 为base64编码的json字符串,字段内容可参照woocoo项目文档说明.

例如:
```
conn, _ := grpc.Dial("polaris://woocoo/helloworld.Greeter", grpc.WithInsecure(), grpc.WithResolvers(&resolverBuilder{}))
```