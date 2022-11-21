# OpenAPI generator for woocoo

本包提供了基于[OpenAPI 3.0](https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.0.md)生成Go代码的一系列工具,用于服务WooCoo的Web项目.
来帮助开发者快速的生成基于OpenAPI 3.0的API服务.

对于像支持partial class的语言(.Net/Go) ,开发者可以很方便的生成的代码中添加自己的代码.

WooCoo选择Gin作为Web框架,因此本包生成的服务端代码也是基于Gin的,Gin项目也可以使用.


## 概览

我们采用了OpenAPI 提供的例子`perstore.ymal`来演示本包的使用方法.[查看该文档](./internal/integration/petstore.yaml).
> 在该文档之上加入一些以`x-go`为前缀的扩展属性

安装woocoo cli
```
go install github.com/tsingsun/woocoo/cmd/woco
woco oasgen -c  ./oasgen/internal/integration/config.yaml
```
接下来让我们看下配置文件内容:

```yaml
# openapi 文件,可以是YAML或JSON格式
spec: "petstore.yaml"
# 生成文件位置,
target: "petstore"
# 期望的包路径,默认同target,可指定相对路径或go包名全路径
package: "petstore"
# 类型映射,
models:
  UUID:
    model: github.com/google/uuid.UUID
```

生成的代码例子可以参考[petstore](./internal/integration/petstore)

## 生成代码结构

首先在指定包名下,会生成以下文件:

- interface.go: 根据Operation生成的接口定义,同时会生成一个未实现的结构体,用于快速的生成一个可运行的服务.
- model.go: 根据Schema生成的数据模型定义
- xx_tag.go: 以Tag为单位生成Operation的Reqeust和Response定义 
- server目录: 服务端相关代码
    server.go: 生成的服务端代码
    validator.go: 生成的数据验证代码,针对OpenAPI的pattern正规做了自定义支持

### 请求及响应

### 请求

根据Sepc的设定,默认生成的请求代码按Path分成Uri,Header,Cookie,Body等,然后分别对各部分字段验证,在此采用了gin使用的[validator](https://github.com/go-playground/validator).
一般不需要另行再编写针对请求的代码.

- Auth验证:(TODO)

### 响应

在Server端代码中,Handler同样根据Spec定义响应结果.

- 200: 以200为成功的响应,返回的数据按http accept进行序列化.
- 错误: 错误处理采用ErrorHandler,只是封装了错误信息进行返回.

> 支持的序列化格式: json,xml,yaml,toml

## 编写服务端代码

1. 通过实现interface.go中定义的接口实现服务端代码.
2. 将服务实现注册至路由,由于每个系统的错误代码并不相同,因此.生成的代码并不定义错误格式. 可自行实现. 如例子使用WooCoo Web内置的`ErrorHandler`来处理错误.

```
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

本包支持自定义模板,通过`-t`参数指定模板目录,模板目录下的文件会覆盖默认模板.参数格式:

```shell
../woco oasgen -c ./internal/integration/config.yaml -t ./internal/integration/template
```
```shell
../woco oasgen -c ./internal/integration/config.yaml -t \
  file=./internal/integration/template/server.tmpl,dir=./oasgen/internal/integration/template2
```
## 开发测试

测试工具采用[swagger-editor](https://github.com/swagger-api/swagger-editor)来做为客户端UI.

swagger-editor安装:
```
docker pull swaggerapi/swagger-editor
docker run -d -p 80:8080 swaggerapi/swagger-editor
```
安装后,可直接把openapi文件拖拽到swagger-editor中,确定好Server地址后,然后启动服务端程序进入测试,可实时调整openapi,非常方便.

## 对比

- [openapi-generator](https://openapi-generator.tech): 
  - 使用java的开发,有很多插件.生成代码质量较低,不能直接使用,模型或服务端代码过于简单,需要自己扩展.
  - 没有interface接口声明,不够灵活.
- [oapi-codegen](https://github.com/deepmap/oapi-codegen)
  - 具有interface接口声明,但接口并没有定义请求与响应涉及的请求响应模型及验证整合,代码量仍然稍大.
  - 采用了模板化.