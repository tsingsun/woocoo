# gRPC

woocoo提供了配置化的gRPC及相关组件，可以让你的服务更加灵活，更加简单。

## 服务端

```yaml
grpc:
  server:
    addr: :20000
    namespace: woocoo # 命名空间,主要用于在服务发现使用
    version: "1.0"
    registryMeta: # 服务发现元数据
      key1: value1
      key2: value2
    engine:
      - keepalive:
          time: 1h
      - tls:
          sslCertificate: ""
          sslCertificateKey: ""
      - unaryInterceptors:          
          - accessLog:
              timestampFormat: "2006-01-02 15:04:05"
          - recovery:
          - auth:
              signingAlgorithm: HS256
              realm: woocoo
              secret: 123456
              privKey: config/privKey.pem
              pubKey: config/pubKey.pem              
      - streamInterceptors:
```

### 拦截器

内置配置:
- keepalive: 连接存活检测,复杂网络环境中需要注意配置
- accessLog: 访问日志
- recovery:
- auth: 基本的JWT验证支持

如果使用其他拦截器,可在代码中使用Option的方式传入.

## 服务发现与治理

woocoo项目中的服务发现与治理目前是GRPC服务中使用.支持的服务发现方式有:

### etcd

实现了较基本的服务注册与发现功能.可通过woocoo直接使用

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
    scheme: etcd
    ttl: 600s
    # 同scheme
    etcd:
      tls:
        sslCertificate: ""
        sslCertificateKey: ""
      endpoints:
        - 127.0.0.1:12379
      dial-timeout: 3s
      dial-keep-alive-time: 3s
```

### polaris

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
      # 目前直接使用polaris自带的简易配置,后续如果配置复杂的话提供配置文件位置方式
      global:
        serverConnector:
          addresses:
            - 127.0.0.1:8091
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

#### 服务端与客户端配置

在元数据节中,由于polaris区分SrcMetadata及DstMetadata,所以需要在客户端和服务端配置中分别配置.

而在woocoo的Registry组件并未这样区分,因此在metadata中配置前缀来对应,使得可以使用Polaris的治理功能.

`dst_`前缀=>对应DstMetadata;`src_`前缀=>对应SrcMetadata,`headerPrefix`=>对应Grpc HeaderPrefix