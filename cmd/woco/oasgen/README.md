# OpenAPI generator for woocoo

本包提供了基于[OpenAPI 3.0](https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.0.md)生成Go代码的一系列工具,用于服务WooCoo的Web项目.

WooCoo选择Gin作为Web框架,因此本包生成的服务端代码也是基于Gin的.

## 概览

我们采用了OpenAPI 提供的例子`perstore.ymal`来演示本包的使用方法.[查看该文档](./internal/integration/petstore.yaml).

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

- interface.go: 根据Operation生成的接口定义
- model.go: 根据Schema生成的数据模型定义
- xx_tag.go: 以Tag为单位生成Operation的Reqeust和Response定义 
- server/server.go: 生成的服务端代码

### 请求及响应

默认生成代码,使用了gin的验证方式.

## 编写服务端代码

通过实现interface.go中定义的接口实现服务端代码.然后注册到路由中即可.

```
	router := gin.Default()
	imp := &Server{}
	server.RegisterUserHandlers(router, imp)
	server.RegisterStoreHandlers(router, imp)
	server.RegisterPetHandlers(router, imp)
```