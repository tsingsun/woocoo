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
```go
logger := log.Component("component-name")
logger.Ctx(ctx).Info("hello world")
``` 

配置结构如下:

```yaml
log:
  disableTimestamp: false # 是否禁用时间戳
  disableErrorVerbose: false # 是否禁用错误详细信息
  callerSkip: 1 # 跳过的调用层级
  # 异步日志配置,启用后日志写入通过 channel 异步发送,减少 I/O 阻塞
  async:
    channelBuffer: 1024
  # 单日志组件,不需要复杂日志记录时一般采用sole
  cores:
    - level: debug
      disableCaller: true
      disableStacktrace: true
      encoding: json #json console 两种格式
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

## 异步日志

启用异步后,日志写入通过 channel 发送到后台 worker goroutine,调用方不阻塞于 I/O.

**工作原理:**
- `Check()`(级别过滤、采样)保持同步,避免将注定丢弃的日志入队
- `Write()` 非阻塞,channel 满时直接丢弃,不影响业务代码
- `Sync()` 停止 worker,排空所有 pending entries,刷新底层 writer

**配置:**
```yaml
log:
  async:
    channelBuffer: 1024  # channel 缓冲大小,默认 1024
```

**注意事项:**
- 程序退出前需确保调用 `Sync()` 刷出日志.使用 `App.Run()` 时会自动处理
- 异步模式下,日志到达输出端存在微小延迟
- 开发调试时建议关闭异步,确保日志即时可见

**性能:** 基准测试显示异步模式比同步模式快约 2.5 倍,在 I/O 密集型场景优势更明显.

### 时间格式

时间格式的配置是相对特殊的,作用于zap.Time相似的方法,`timeEncoder`支持配置如下:

- "rfc3339nano" 或 "RFC3339Nano" 对应 RFC3339NanoTimeEncoder.
- "rfc3339" 或 "RFC3339" 对应 RFC3339TimeEncoder.
- "iso8601" 或 "ISO8601" 对应 ISO8601TimeEncoder.
- "millis" 对应 EpochMillisTimeEncoder.
- "nanos" 对应 EpochNanosEncoder.

在不满足时,可以自定义格式:

```yaml
  timeEncoder:
	layout: 06/01/02 03:04pm
```

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
