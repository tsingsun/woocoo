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

```yaml
grpc:
  server:
    addr: :20000
    namespace: /woocoo/service
    version: "1.0"
    registryMeta:
      key1: value1
      key2: value2
  registry:
    scheme: polaris
    ttl: 600s
    polaris:
      # 目前直接使用polaris自带的简易配置,后续如果配置复杂的话提供配置文件位置方式,配置文件优先于内嵌配置,同时只能2选1.
      global:
        serverConnector:
          addresses:
            - 127.0.0.1:8091
      # 采用配置文件方式.此时内嵌配置无效
      configFile: etc/polaris.yaml      
  client:
    target:
      namespace: woocoo
      serviceName: helloworld.Greeter
      metadata:
        version: "1.0"
        dst_location: amoy
        src_tag: tag1
        headerPrefix: "head1,head2"
```

### 路由配置

在元数据节中,由于polaris区分SrcMetadata及DstMetadata,所以需要在客户端和服务端配置中分别配置.

而在woocoo的Registry组件并未这样区分,因此在metadata中配置前缀来对应,使得可以使用Polaris的治理功能.

`dst_`前缀=>对应DstMetadata;`src_`前缀=>对应SrcMetadata,`headerPrefix`=>对应Grpc HeaderPrefix