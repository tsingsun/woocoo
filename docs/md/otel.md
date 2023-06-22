---
title: 可观测性
---

# 可观测性

OpenTelemetry 是 CNCF 的一个可观测性项目，旨在提供可观测性领域的标准化方案，解决观测数据的数据模型、采集、处理、导出等的标准化问题，
提供与三方 vendor 无关的服务。主要核心模块为链接跟踪和程序运行指标收集.

本项目使用了OpenTelemetry实现了链路追踪。

## 配置使用

支持以配置文件的方式引入,默认使用了全局的TraceProvider与MeterProvider.

在程序配置文件中`otel`节点为链接追踪相关的配置：
```yaml
otel:
  # trace,当设置为""时,则使用默认的noop
  traceExporter: "stdout"
  # metric
  metricExporter: "stdout"
```

- 在web服务采用中间件集成使用. 
```yaml
web:
  engine:
    routerGroups:
    - default:
        handleFuncs:
        - otel:
```

- 在grpc服务采用拦截器集成使用.
```yaml
grpc:
  server:
    interceptors:
    - otel:
```

### 传播

对于trace的上下文传播,如java->go等对透传有特定协议的,我们需要确定采用的传播者. 像常见的传播者有:`b3`,`jaeger`,`aws`,`ot`等,
[详见](https://opentelemetry.io/docs/reference/specification/context/api-propagators/#propagators-distribution).

b3的应用范围比较广泛,因此全局的otel的配置目前支持`b3`的简单配置:
```yaml
otel:
  propagators: "b3"
```

当采用其他或者复杂的初始化时,需要自己在代码中初始化,如:
```go
// 例如可以把替换为其他的传播者
otelcfg := telemetry.NewConfig(otelCnf,telemetry.WithPropagators(b3.New()))
// 或者
otel.SetTextMapPropagator(b3.New())
```

## OTLP

在配置文件中使用OTLP时,我们需要搭建相应的环境,我们以Jeager为例.

### 需求环境

[OTEL Collector 安装](https://opentelemetry.io/docs/collector/getting-started/)
[Jeager 安装](https://www.jaegertracing.io/docs/1.40/getting-started/)

otel连接jeager配置如下:
```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 127.0.0.1:4317
  zipkin:
processors:
  batch:
exporters:
  jaeger:
    endpoint: "127.0.0.1:14250"
    tls:
      insecure: true    
service:
  pipelines:
    traces:
      receivers: [otlp,zipkin]
      processors: [batch]
      exporters: [jaeger]    
```
启动otelcol和jeager后,我们可以通过`http://localhost:16686`访问UI,查看链路追踪.

#### 程序配置

在otel环境启动后,我们需要在我们的woocoo应用配置文件做相应的调整.

```yaml
  # otlp
  traceExporter: "otlp"
  otlp:
    endpoint: "127.0.0.1:4317"  
    client:
      dialOption:
        - tls: 
        - block:
        - timeout: 5s
  metricExporter: "otlp"
```
> metric如果同为otlp,则配置同trace,但在连接层面上为不同的连接.