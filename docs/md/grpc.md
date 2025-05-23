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

```yaml
- unaryInterceptors:          
  - accessLog:
      # 除了基本的字段,还默认支持以下:
      # grpc.start_time,grpc.service,grpc.method,grpc.request.deadline,status,error,latency,peer.address,request,response
      # 请以','分隔
      format: "status,error,latency"
  # 无配置项
  - recovery:
  # 与JWT一致
  - auth:
      # 不需要验证的方法
      exclude: ["/helloworld.Greeter/SayHello"]
      signingAlgorithm: HS256
      realm: woocoo
      secret: 123456
      privKey: config/privKey.pem
      pubKey: config/pubKey.pem              
- streamInterceptors:
  - accessLog:
      # 不需要验证的方法
      exclude: ["/helloworld.Greeter/SayHello"]
      # 除了基本的字段,还默认支持以下:
      # grpc.start_time,grpc.service,grpc.method,grpc.request.deadline,status,error,latency,peer.address
      # 请以','分隔
      format: "status,error,latency"
    # 无配置项
  - recovery:
    # 与unaryInterceptors一致    
  - auth:
```

## 客户端

```yaml
grpc:
  # 对于客户端,配置服务端, 可以自动获取Dial参数.
  server:
    addr: :20000
  client:
    # target段主要配合服务发现,如不需要可去掉
    target:
      namespace: woocoo
      serviceName: helloworld.Greeter
      metadata: 
        version: "1.0"
    dialOption:      
      - block:
      - timeout: 5s
      - tls:
          cert: "x509/server.crt" 
      - unaryInterceptors:
          - otel:
```

在 grpcx.Client 定义了grpc client工具.可以方便通过配置文件创建.但目前的功能还只是快速connection的创建

```go
// 如果指定了server的配置或者使用服务发现, Dial可置为空
client,_ := grpcx.NewClient(cfg)
client.Dial("")
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
    # 节点名与scheme相同
    etcd:
      endpoints:
        - 127.0.0.1:12379
      dial-timeout: 3s
      dial-keep-alive-time: 3s
```

涉及到中型项目的微服务中可采用更加能力强大的服务治理来管理.可参考[服务治理](micro.md)

