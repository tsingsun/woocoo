# WooCoo

[English](README.md) | 🇨🇳中文

## WooCoo简介

WooCoo的定位是一个基于Golang的应用开发框架及工具包,以便开发者通过本工具来开发各种Api应用或RPC服务.

本项目更偏向粘合剂作用,核心组件选取开源流程项目,目前实现的功能:

本工具包提供下列功能:

- 基本组件配置化,多配置文件支持,方便多环境下的开发配置
- 日志组件,支持多日志输出,请参见[Logger](docs/logger.md),
- OpenTelemetry支持,请参见[OpenTelemetry](docs/otel.md)
- web服务,支持GraphQL
- grpc服务
- JWT-based验证

核心组件的选取:

- 日志: [Uber Zap](http://go.uber.org/zap)
- Web路由框架: [gin](http://github.com/gin-gonic/gin)

微服务相关:

- 服务注册与发现: 实现了[etcd](https://github.com/coreos/etcd),留有其他组件扩展的能力

## 其他

联系我: QQ: 21997272

## 致谢

![image](https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.svg)