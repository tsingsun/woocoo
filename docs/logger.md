## 日志

框架日志组件内置了Zap+Rotate组合,采用文件流记录日志.

文件流方式为性能最高的一种方式,满足绝大部分应用场景. 对于日志收集中间件来说文件采集支持也是必备的.

配置结构如下:

```yaml
log:
  # 单日志组件,不需要复杂日志记录时一般采用sole
  sole:
    level: debug
    disableCaller: true
    disableStacktrace: true
    encoding: json
    encoderConfig:
      timeEncoder: iso8601
    outputPaths:
      - stdout
      - "test.log"
    errorOutputPaths:
      - stderr
  # 需要做日志拆分采用tee配置.
  tee:
    - level: debug 
      disableCaller: true
      disableStacktrace: true
      encoding: json
      encoderConfig:
        timeEncoder: iso8601
      outputPaths:
        - stdout
        - "test.log"
      errorOutputPaths:
        - stderr
    - level: warn 
      disableCaller: true
      outputPaths: 
        - "test.log"
      errorOutputPaths:
        - stderr
  # 采用文件流时,轮转配置可方便管理与跟踪日志,可选配置
  rotate:
    maxSize: 1
    maxage: 1
    maxbackups: 1
    localtime: true
    compress: false
```
## sole && tee

内置配置基于Zap的Config对象