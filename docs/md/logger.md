---
title: 日志
---
## 日志

框架日志组件内置了[Uber Zap](http://go.uber.org/zap)+Rotate组合,采用文件流记录日志.

文件流方式为性能最高的一种方式,满足绝大部分应用场景. 对于日志收集中间件来说文件采集支持也是必备的.

为了支持不同场景下的使用,具有多种使用方式:

- 普通日志: 类似go及zap的使用方式.
```go
log.Info("hello world")
```
- 组件日志: 
```go
logger := log.Component("component-name")
logger.Info("hello world")
```
- 上下文日志: 把上下文信息记录到日志
  > 每一次调用Ctx创建ContextLogger后调用日志记录方法,都会回收ContextLogger,因此应避免.
```go
logger := log.Component("component-name")
logger.Ctx(ctx).Info("hello world")
// 不可以使用下面的方式
clog := logger.Ctx(ctx)
clog.Info("hello world")
clog.Info("hello world1")
``` 

配置结构如下:

```yaml
log:
  disableTimestamp: false # encoder text 时,是否显示时间戳
  disableErrorVerbose: false # encoder text 时,是否显示错误详情
  callerSkip: 1 # 跳过的调用层级
  # 单日志组件,不需要复杂日志记录时一般采用sole
  cores:
    - level: debug
      disableCaller: true
      disableStacktrace: true
      encoding: json #json console text 三种格式
      encoderConfig:
        timeEncoder: iso8601 # 默认值
      # outputPaths 日志输出路径,支持stdout,stderr,文件路径
      # default: stderr. 使用的zap的默认值.
      outputPaths:
        - stdout
        - "test.log"
      errorOutputPaths:
        - stderr
  # 采用文件流时,轮转配置可方便管理与跟踪日志,可选配置;
  rotate:
    maxSize: 1
    maxage: 1
    maxbackups: 1
    localtime: true
    compress: false
```

`rotate`可只保留key,不配置值,则使用默认值.默认值如下:

- MaxSize: 单文件最大大小, 100MB
- MaxAge: 文件保留天数, 不限制
- MaxBackups: 保留文件个数, 不限制
- LocalTime: false, 使用UTC时间
- Compress: false, 不压缩

## mulit-logger

```yaml
  # 日志组件,需要复杂日志记录时一般采用multi
  cores:
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
```
内置配置基于Zap的Config对象

## Web访问日志

在web服务中,经常需要记录访问日志,框架提供了一个中间件,用于记录访问日志,同时搭配recovery中间件来处理panic错误.

```yaml
web:
  server:
    addr: 0.0.0.0:33333
  engine:
    routerGroups:
      - default:
          middlewares:
            - accessLog:
                exclude:
                  - /healthCheck
```

Error的处理: 
  - 对于内部错误时,记录类型为Error
  - 对于公共错误,如404,500等,记录类型为Info

Panic的处理: 额外记录stacktrace

## grpc服务端访问日志

grpc访问日志以拦截器形式实现支持,搭配recovery拦截器来附加panic错误.

```yaml
grpc:
  server:
    engine:
      - unaryInterceptors:
          - accessLog:
              timestampFormat: "2006-01-02 15:04:05"
          - recovery:
```

Error的处理根据grpc的状态码来判断: 详见`interceptor.DefaultCodeToLevel`函数

Panic的处理: 额外记录stacktrace

## 结合标准库

在使用某些第三方库时.如果支持设置`io.Writer`,则可转化为woocoo的日志.log库内置了实现`io.Writer`类,可直接使用.

```go
import (
	"log"
	wclog "github.com/tsingsun/woocoo/pkg/log"
)
w := &wclog.Writer{
    Log:   wclog.Global().Logger(),
	// Level 默认级别
    Level: zap.InfoLevel,
}
log.SetOutput(w)
// 或者使用Component时,可以直接使用
logger = wclog.Component("web")
log.SetOutput(logger.Logger().IOWriter(zapcore.DebugLevel))
```

除了转换功能外,还可通过识别如`[{level}]`文本提取日志级别,并记录到日志中.

支持的文本有: debug,info,warn,error,fatal,panic, 例如:
```go
// 使用标准库记录
log.Print("[debug]hello world")
log.Print("[DEBUG]hello world")
log.Println("Web [info] hello world")
```
