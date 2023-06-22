# gRPC

woocoo提供了配置化的gRPC及相关组件，可以让你的服务更加灵活简单,后以GRPC为基石来扩展整个微服务体系.

## 服务端

```yaml
grpc: # 可选的顶级节点名
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
          cert: "" # 文件地址,可相当程序启动目录的相对地址
          key: "" # 文件地址,可相当程序启动目录的相对地址
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

## 客户端

```yaml
grpc:
  client:
    target:
      namespace: woocoo
      serviceName: helloworld.Greeter
      metadata: 
        version: "1.0"
    dialOption:
      - insecure:
      - block:
      - timeout: 5s
      - tls:
          cert: "x509/server.crt" 
      - unaryInterceptors:
          - otel:
```

在 grpcx.Client 定义了grpc client工具.可以方便通过配置文件创建.但目前的功能还只是快速connection的创建

```
// 如果指定了service的配置,可自动获取
grpcx.NewClient(cfg).Dial("127.0.0.1:8080")
```

## 服务发现

woocoo项目中的可简单的使用服务发现:

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

涉及到中型项目的微服务中可采用更加能力强大的服务治理来管理.可参考[服务治理](micro.md)

