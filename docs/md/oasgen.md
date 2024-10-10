---
title: OpenAPI3导入
---

## OpenAPI3 generator for woocoo

本包提供了基于[OpenAPI 3.0](https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.0.md)生成Go代码的一系列工具,用于服务WooCoo的Web项目.
来帮助开发者快速的生成基于OpenAPI 3.0的API服务.

对于像支持partial class的语言(.Net/Go) ,开发者可以很方便的生成的代码中添加自己的代码.

WooCoo选择Gin作为Web框架,因此本包生成的服务端代码也是基于Gin的,Gin项目也可以使用.

## 概览

我们采用了 OpenAPI 提供的例子`perstore.ymal`来演示本包的使用方法.[查看该文档](https://github.com/tsingsun/woocoo/blob/master/cmd/woco/oasgen/internal/integration/petstore.yaml).
> 在该文档之上加入一些以`x-go`为前缀的扩展属性

安装woocoo cli
```
go install github.com/tsingsun/woocoo/cmd/woco
```

命令行参数说明:

```shell
NAME:
   woco oasgen - a tool for generate woocoo web code from OpenAPI 3 specifications

USAGE:
   woco oasgen [command options] [arguments...]

OPTIONS:
   --config value, -c value                                   配置文件位置
   --template value, -t value [ --template value, -t value ]  扩展模板文件,可以指定文件夹或文件
   --client                                                   生成客户端代码.
   --help, -h                                                 show help
```

```shell
# 生成服务端代码, 一般服务端代码较为复杂,使用额外的配置文件.
woco oasgen -c ./oasgen/internal/integration/config.yaml
# 生成客户端代码, 默认当前目录中寻找./opanapi.yaml文件; 以当前文件夹名做为包名
woco oasgen --client
# 生成客户端代码,指定配置.
woco oasgen -c ./oasgen/internal/integration/config.yaml --client
```

:::note
如果在monorepo中,需要指定`package`参数.遇到`root package or module was not found for`错误时,请检查`package`参数是否正确.
:::

接下来让我们看下配置文件内容:

```yaml
# openapi 文件,可以是YAML或JSON格式
spec: "petstore.yaml"
# 生成文件位置,
target: "petstore"
# 期望的包路径,默认同target,可指定相对路径或go包名全路径
package: "petstore"
# 外部模板文件,用于一般存放自定义的模板文件
templateDir: "template"
# 类型映射,
models:
  UUID:
    model: github.com/google/uuid.UUID
  # 映射至其他包,使用ref为key,被映射的将不会被生成Struct  
  '#/components/schemas/ApiResponseXX':
    model: github.com/tsingsun/woocoo/cmd/woco/oasgen/internal/integration.ApiResponse
```

生成的代码例子可以参考[petstore](https://github.com/tsingsun/woocoo/tree/master/cmd/woco/oasgen/internal/integration/petstore)

## 生成代码结构

首先在指定包名下,会生成以下文件:

- interface.go: 根据 Operation 生成的接口定义,同时会生成一个未实现的结构体,用于快速的生成一个可运行的服务.
- model.go: 根据 Schema 生成的数据模型定义
- tag_xxx.go: 以 Tag 为单位生成 Operation 的 Reqeust 和 Response 定义 
- handler.go: 服务端路由及 handler 代码
- validator.go: 服务端的数据验证代码,针对 OpenAPI 的 pattern 正则做了自定义支持
- client.go: 客户端基础代码
- api_xxx.go: 以 Tag 为单位生成的客户端调用代码

在生成的结构体字段并没有按照文档顺序,而是内部根据名称顺序做排序,这主要是由于 OpenAPI 的解析库的采用无序 Map 的原因.

### 请求及响应

### 请求

根据 Sepc 的设定,默认生成的请求代码按参数类型定义 `in` 分成 Path, Header, Cookie, Query, Body 等然后分别对各部分字段验证,在此采用了 gin 使用的[validator](https://github.com/go-playground/validator).
一般不需要另行再编写针对请求的代码.

:::warning

由于 Gin Binding 验证器的限制,无法将全部参数定义在一个结构体中,如果具有多种不同类型的参数,采用了分组的方式,每个分组对应一个结构体以分别调用`BindXXX`方法绑定请求参数.

:::

请求参数类型及定义:

- Path: 以 Gin`/path/:` 的方式定义,在生成的代码中,会将 `{param}` 替换为 `%v` 的形式,以便于使用 `fmt.Sprintf` 进行格式化.

请求参数验证:

- 输入验证: 通过 Request 代码,已经内置了 openapi3 所描述的常用的格式验证.
  - String/Number的最大值,最小值,长度等验证
  - Dates, Times, and Duration
  - Email Addresses
  - Hostnames
  - IP Addresses
  - Regular Expressions
  - 支持通过`x-go-tag-validator`扩展验证属性,具体可参考 validator 的表达式.
  - Enum: 采用 OneOf 对 Enum 类型验证. 目前只支持 string 类型的 Enum.  由于在 Spec 中无法知道对应的 Model 的类型, 因此不再针对请求参数去生成 Enum  类型.
- Auth验证: 这部分的未做过多的代码生成,需要结合中间件配置.
  内置支持的 JWT, KeyAuth 验证.要将不验证路径配置入 `exclude` 中

### 响应

在 Server 端代码中,Handler 同样根据 Spec 定义响应结果.

需要说明的是常见的模式中会生成 XXXResponse 对象来封装返回,这种模式的好处就是当接口变化时,兼容性很高.
而这是由接口定义决定的,即接口文件怎么定义我们就怎么生成返回对象.因此针对响应结果不再进行包装.

- 200: 以200为成功的响应,返回的数据按 http accept 进行序列化.
- 错误: 错误处理采用 ErrorHandler,只是封装了错误信息进行返回.

> 支持的序列化格式: json,xml,yaml,toml

## 扩展

- 自定义 Tag: 通过 `x-go-tag` 可以自定义 Tag.
- 自定义验证: 通过 `x-go-tag-validator` 可以自定义验证器.
- 忽略生成: 通过 `x-go-codegen-ignore` 可以控制 `Operation` 是否生成, 以自行实现接口定义及逻辑.

## 编写客户端代码

在生成的代码如果不能满足需求, 比如需要传入动态参数,或者需要自定义的请求头,这时候就需要利用拦截器能力.

拦截器的定义为 `func(ctx context.Context, req *http.Request) error`,如果返回错误,则会终止请求.

拦截器的执行点在 `client.APIClient.Do` 方法中,在执行请求前,会依次执行拦截器,
```go
// client是指向的包名
cli := client.NewAPIClient(client.Config)
cli.AddInterceptor(func(ctx context.Context, req *http.Request) error {
    req.Header.Add("X-Interceptor", "true")
    req.Header.Add("X-Interceptor-Value", "1")
    return nil
})
```

## 编写服务端代码

1. 通过实现 interface.go 中定义的接口实现服务端代码.
2. 将服务实现注册至路由,由于每个系统的错误代码并不相同,因此.生成的代码并不定义错误格式. 可自行实现. 如例子使用 WooCoo Web 内置的 `ErrorHandler` 来处理错误.

```go
type Server struct {
	petstore.UnimplementedStoreServer
	petstore.UnimplementedPetServer
	petstore.UnimplementedUserServer
}

func main() {
	router := gin.Default()
	router.Use(handler.ErrorHandle().ApplyFunc(nil))
	imp := &Server{}
	server.RegisterUserHandlers(router, imp)
	server.RegisterStoreHandlers(router, imp)
	server.RegisterPetHandlers(router, imp)
}	
```

## 自定义模板

本包支持自定义模板,通过 `-t` 参数指定模板目录,模板目录下的文件会覆盖默认模板.参数格式:

```shell
../woco oasgen -c ./internal/integration/config.yaml -t ./internal/integration/template
```
```shell
../woco oasgen -c ./internal/integration/config.yaml -t \
  file=./internal/integration/template/server.tmpl,dir=./oasgen/internal/integration/template2
```
## 开发测试

测试工具采用[swagger-editor](https://github.com/swagger-api/swagger-editor)来做为客户端UI.

swagger-editor 安装:
```
docker pull swaggerapi/swagger-editor
docker run -d -p 80:8080 swaggerapi/swagger-editor
```
安装后,可直接把 openapi 文件拖拽到 swagger-editor 中,确定好 Server 地址后,然后启动服务端程序进入测试,可实时调整 openapi,非常方便.

## 对比

- [openapi-generator](https://openapi-generator.tech): 
  - 使用 java 的开发,有很多插件.生成代码质量较低,不能直接使用,模型或服务端代码过于简单,需要自己扩展.
  - 没有 interface 接口声明,不够灵活.
- [oapi-codegen](https://github.com/deepmap/oapi-codegen)
  - 具有 interface 接口声明,但接口并没有定义请求与响应涉及的请求响应模型及验证整合,代码量仍然稍大.
  - 采用了模板化.