---
id: guide
title: 简介
---

# 关于 WooCoo

**WooCoo**（音：武库）是一个基于 Go 的应用开发框架和工具包，旨在帮助开发者快速构建高性能的 Web API 和微服务应用。

[![Language](https://img.shields.io/badge/Language-Go-blue.svg)](https://golang.org/)
[![codecov](https://codecov.io/gh/tsingsun/woocoo/branch/main/graph/badge.svg)](https://codecov.io/gh/tsingsun/woocoo)
[![Go Report Card](https://goreportcard.com/badge/github.com/tsingsun/woocoo)](https://goreportcard.com/report/github.com/tsingsun/woocoo)
[![Build Status](https://github.com/tsingsun/woocoo/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/tsingsun/woocoo/actions?query=branch%3Amain)
[![Release](https://img.shields.io/github/release/tsingsun/woocoo.svg?style=flat-square)](https://github.com/tsingsun/woocoo/releases)
[![GoDoc](https://pkg.go.dev/badge/github.com/tsingsun/woocoo?status.svg)](https://pkg.go.dev/github.com/tsingsun/woocoo?tab=doc)

## 设计理念

WooCoo 的核心理念是**"优秀的粘合剂"**——我们不重复造轮子，而是将业界优秀的开源组件集成起来，提供一个统一的、约定式的开发框架，让你专注于业务逻辑的开发。

## 核心特性

### 🚀 快速开发
- **约定式配置**：基于约定优于配置的原则，减少样板代码
- **Gin 引擎集成**：完全兼容 Gin 的路由和中间件生态
- **GraphQL 支持**：内置 GraphQL 支持，可与 [Ent ORM](https://entgo.io/) 无缝集成

### 🌐 Web 开发
- 基于 [Gin](https://github.com/gin-gonic/gin) 的高性能 Web 框架
- 内置常用中间件：JWT 认证、CORS、访问日志、错误处理等
- 灵活的中间件配置和加载顺序控制
- 支持 HTTP/HTTPS 服务

### 🔌 微服务支持
- **gRPC 服务**：内置 gRPC Server，轻松构建微服务
- **服务注册与发现**：
  - [etcd v3](https://etcd.io/)：服务注册与发现
  - [Polaris](https://github.com/polarismesh/polaris)：服务治理与发现
- **OpenTelemetry**：原生支持分布式追踪和可观测性

### ⚙️ 工程化实践
- **灵活配置管理**：基于 [Koanf](https://github.com/knadh/koanf) 的配置系统，支持 YAML、环境变量、多文件合并等
- **高性能日志**：基于 [Zap](https://github.com/uber-go/zap) 的结构化日志，支持日志轮转
- **缓存支持**：内置 Redis 和内存缓存支持

### 🛠️ 开发工具
- **WoCo CLI**：代码生成工具，快速创建项目模板
- **OpenAPI 3.0 代码生成**：从 OpenAPI 规范生成服务端代码
- **Ent ORM 集成**：支持 Ent 代码生成

## 技术栈

WooCoo 的核心组件来自以下优秀项目：

| 组件 | 技术栈 | 用途 |
|------|--------|------|
| Web 引擎 | [Gin](https://github.com/gin-gonic/gin) | HTTP 路由和处理 |
| 配置管理 | [Koanf](https://github.com/knadh/koanf) | 灵活的配置加载 |
| 日志系统 | [Zap](https://github.com/uber-go/zap) | 高性能结构化 日志 |
| gRPC | [gRPC-Go](https://github.com/grpc/grpc-go) | RPC 通信 |
| 缓存 | [go-redis](https://github.com/redis/go-redis) | Redis 客户端 |
| JWT | [jwt/v5](https://github.com/golang-jwt/jwt) | 身份认证 |

## 性能表现

WooCoo 在 Gin 的基础上进行了优化，在 Web 场景下表现出色的性能：

| Benchmark | 总吞吐量 | 单次延迟 | 堆内存 | 分配次数 |
|-----------|---------|----------|--------|----------|
| WooCoo Web | 564,612 | 2633 ns/op | 1103 B/op | 5 allocs/op |
| Gin Default | 81,198 | 14,418 ns/op | 354 B/op | 13 allocs/op |
| Gin MockLogger | 423,054 | 2747 ns/op | 221 B/op | 8 allocs/op |

> 测试环境：macOS, Intel Core i7-9750H @ 2.60GHz

## 快速开始

```bash
# 创建项目
mkdir myapp && cd myapp
go mod init myapp
go get github.com/tsingsun/woocoo

# 或使用 CLI 工具快速创建项目
go install github.com/tsingsun/woocoo/cmd/woco@latest
woco init -p myapp -m cache,web,grpc,otel -t ./myapp
```

查看 [快速开始](./quickstart) 了解更多详情。

## 示例项目

- [WooCoo 示例项目](https://github.com/tsingsun/woocoo-example)：完整的示例代码和最佳实践

## 社区与支持

- **GitHub**: [https://github.com/tsingsun/woocoo](https://github.com/tsingsun/woocoo)
- **Discord**: [加入社区](https://discord.gg/358d5uth)
- **Stack Overflow**: [提问标签](https://stackoverflow.com/questions/tagged/woocoo)
- **GoDoc**: [API 文档](https://pkg.go.dev/github.com/tsingsun/woocoo)

## 许可证

WooCoo 采用 [MIT 许可证](https://github.com/tsingsun/woocoo/blob/main/LICENSE)

---

**准备好开始了吗？** 前往 [快速开始](./quickstart) 了解如何使用 WooCoo 构建你的第一个应用！
