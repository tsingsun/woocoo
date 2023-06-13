# 服务治理

在应用架构从单体架构向微服务架构演进的过程中，服务治理是一个重要的话题。服务治理的目标是为了解决服务的发现、注册、路由、负载均衡、熔断、限流、降级、监控、追踪等问题。
常见的治理平台如: [腾讯polarismesh](https://polarismesh.cn),[阿里nacos](https://nacos.io/zh-cn/)

woocoo可以通过配置文件方式来实现服务治理中间件的集成.

## polarismesh

腾讯开源的北极星是一个支持多语言、多框架的云原生服务发现和治理中心， 解决分布式和微服务架构中的服务可见、故障容错、流量控制和安全问题。

```
go get github.com/tsingsun/woocoo/polarismesh
```  
现使用polaris默认配置的话所需的配置项很少,因此直接在woocoo配置文件中集成了.

支持能力有:

- [x] 服务注册及发现
- [x] 动态路由
- [x] 负载均衡
- [x] 节点熔断
- [x] 访问限流:部分能力
    - 根据请求方法进行限流.
- [] 配置管理
- [x] 可观测性

可以查看[示例](https://github.com/tsingsun/woocoo-example/tree/main/grpc/polaris)

## 接入配置

在woocoo微服务`registry`配置节中,指定相应的北极星网格:

```yaml
grpc:
  registry:
    scheme: polaris # 必须指定为polaris
    global: true #该节点配置是否做为全局配置.
    ttl: 30s # 心跳时间,[0-60]s
    polaris:
      ... # 该节占下同北极星网格本身的配置  

# 也可以指定为一个独立的配置节,使用独立的配置文件,如下:
grpc2:
  registry:
    scheme: polaris # 仍然需要指定为polaris
    ref: registry # 以root为起点的配置节路径,对于引用的注册中心,会默认将第一个初始化全局配置.
registry:
  ttl: 30s # 心跳时间,[0-60]s
  polaris:
    # 采用配置文件方式.此时内嵌配置无效
    configFile: etc/polaris.yaml  
```

### client

woocoo微服务中的客户端可以采用标准的GRPC客户端.但更简单的方式是使用woocoo扩展的`grpcx.client`,不管使用哪种微服务中心,都可以使用该客户端.

配置如下:

```yaml
grpc:
  registry:
    scheme: polaris
    global: true
    ttl: 30s # 心跳时间,[0-60]s
    polaris:
      global:
        serverConnector:
          addresses:
            - 127.0.0.1:8091
  client:
    target: # 该节相当于Dial的目标地址信息
      namespace: default # 服务命名空间
      serviceName: helloworld.Greeter # 服务名称
      metadata:        
        route: true # 是否启用动态路由,默认为false
        header_location: amoy #                
```

在`metadata`元数据节中,polaris区分SrcMetadata及DstMetadata,所以需要在客户端和服务端配置中分别配置.
而在woocoo的Registry组件并未这样区分,因此在metadata中配置前缀来对应,使得可以使用Polaris的治理功能.

`header_`前缀的参数在会在对应Polaris中的请求头参数.

在客户端与服务端的通信过程中,grpcx.client会将配置中的`metadata`作为元数据传递给服务端,服务端可以根据元数据来进行路由,限流等操作.规则如下:

- namespace,serviceName默认传递
- header_{key}:{value}以 key:value 加入outgoing context.

在配置文件中,为固化参数.如果需要使用动态参数,如用户ID,可使用interceptor来实现.

#### grpcx.client

初始化方式没有特殊点.

```go
client := grpcx.NewClient(app.AppConfiguration().Sub("grpc"))
// 可以传空.
conn, err := client.Dial("")
// 或者
conn, err := client.Dial("polaris://helloworld.Greeter")
```

#### 原生client

需要指定resolver和负载均衡配置.当然resolver也可以全局注入. 而grpcx.client会自动初始化以支持动态配置.

```go
conn, err := grpc.Dial(scheme+"://routingTest/helloworld.Greeter?route=true",		
			grpc.WithResolvers(&resolverBuilder{}),
			grpc.WithDefaultServiceConfig(`{ "loadBalancingConfig": [{"polaris": {}}] }`),
		)

```