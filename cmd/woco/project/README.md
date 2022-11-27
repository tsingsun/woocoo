# Initial Project Tool

`woco init`是一个通过模板快速初始化项目的工具,可通过`woco init -h`查看

参数说明:

- package: 项目完整包名
- target: 项目文件位置
- modules: 支持的模块名:
  - web: web项目
  - grpc: grpc项目
  - otel: opentelemetry支持
  - cache: 缓存支持

## 生成说明

- Web项目没有实际的路由注册,需要进一步开发.对于OpenAPI项目还可以通过`woco oasgen`命令生成.
- GRPC项目未注册服务,需要进一步开发

