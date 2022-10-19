# 链接追踪

OpenTelemetry 是 CNCF 的一个可观测性项目，旨在提供可观测性领域的标准化方案，解决观测数据的数据模型、采集、处理、导出等的标准化问题，
提供与三方 vendor 无关的服务。

本项目使用了OpenTelemetry实现了链路追踪。

## 配置使用

支持以配置文件的方式引入,默认使用了全局的TraceProvider与MeterProvider.

在程序配置文件中`otel`节点为链接追踪相关的配置：
```yaml
otel:
  # stdout,endpoint地址,当设置为""时,则使用默认的noop
  traceExporterEndpoint: "stdout" or ":4317"
  metricExporterEndpoint: "stdout" or ":4317"
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