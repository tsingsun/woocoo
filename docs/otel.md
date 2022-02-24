# OpenTelemetry

OpenTelemetry 是 CNCF 的一个可观测性项目，旨在提供可观测性领域的标准化方案，解决观测数据的数据模型、采集、处理、导出等的标准化问题，提供与三方 vendor 无关的服务。

本项目将支持OpenTelemetry中的stdout,otlp两种Provider.已支持的项目:

- web服务,可通过配置或自定义的方式.
  - 配置方式
    ```yaml
    web:
      engine:
        routerGroups:
        - default:
            handleFuncs:
            - otel:
                # stdout,endpoint地址,当设置为""时,则使用默认的noop
                traceExporterEndpoint: "stdout" or ":4317" or ""
                metricExporterEndpoint: "stdout" or ":4317" or ""
    ```   