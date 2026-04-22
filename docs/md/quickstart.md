---
id: quickstart
title: 快速开始
---

# 快速开始

在本教程中，你将使用 WooCoo 快速构建一个包含 Web API 和 gRPC 服务的完整应用。

## 前置要求

- **Go 1.24+**：[下载并安装 Go](https://golang.org/doc/install)
- **基本 Go 知识**：了解 Go 模块和包管理

:::tip 提示
如果你是 Go 新手，建议先阅读 [Go 官方教程](https://go.dev/learn/)。
:::

## 方法一：手动创建项目（推荐学习）

### 1. 初始化项目

```bash
# 创建项目目录
mkdir myapp && cd myapp

# 初始化 Go 模块
go mod init myapp

# 安装 WooCoo
go get github.com/tsingsun/woocoo
```

### 2. 创建项目结构

```bash
mkdir -p cmd/etc
```

项目结构如下：

```
myapp/
├── cmd/
│   ├── etc/
│   │   └── app.yaml    # 配置文件
│   └── main.go         # 程序入口
├── go.mod
└── go.sum
```

### 3. 创建配置文件

创建 `cmd/etc/app.yaml`：

```yaml
# 应用基本信息
namespace: default
appName: myapp
version: 1.0.0
development: true  # 开发模式

# Web 服务配置
web:
  server:
    addr: :8080  # 服务监听地址
```

### 4. 编写主程序

创建 `cmd/main.go`：

```go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web"
	"log"
)

func main() {
	// 加载配置（自动从 cmd/etc/app.yaml 读取）
	app := woocoo.New()

	// 创建 Web 服务
	webSrv := web.New(web.WithConfiguration(conf.Global().Sub("web")))
	
	// 注册路由
	webSrv.Router().GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello, WooCoo!",
			"version": conf.Global().GetString("version"),
		})
	})
	
	// 健康检查端点
	webSrv.Router().GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 注册服务到应用
	app.RegisterServer(webSrv)

	// 启动应用
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
```

### 5. 运行应用

```bash
# 进入 cmd 目录
cd cmd

# 运行程序
go run main.go
```

在另一个终端窗口测试：

```bash
# 测试根路径
curl http://localhost:8080/

# 输出: {"message":"Hello, WooCoo!","version":"1.0.0"}

# 测试健康检查
curl http://localhost:8080/health

# 输出: {"status":"ok"}
```

## 方法二：使用 CLI 工具（快速开始）

WoCo CLI 可以帮你快速生成一个包含完整功能的项目模板。

### 1. 安装 CLI

```bash
go install github.com/tsingsun/woocoo/cmd/woco@latest
```

验证安装：

```bash
woco -v
```

### 2. 创建项目

```bash
# 创建包含 Web、gRPC、缓存和 OpenTelemetry 的项目
woco init -p github.com/yourname/myapp -m cache,web,grpc,otel -t ./myapp

# 进入项目目录
cd myapp
```

生成的项目结构：

```
myapp/
├── cmd/
│   ├── etc/
│   │   └── app.yaml      # 配置文件
│   └── main.go           # 程序入口
├── go.mod
├── go.sum
└── README.md
```

### 3. 运行项目

```bash
cd cmd
go run main.go
```

## 下一步

现在你已经成功运行了一个 WooCoo 应用！继续学习以下内容：

- 📚 [配置系统](./conf) - 了解灵活的配置管理
- 🌐 [Web 开发](./gin) - 构建 RESTful API
- 🔌 [gRPC 服务](./grpc) - 构建微服务
- 📊 [日志系统](./logger) - 结构化日志记录
- 🔍 [OpenTelemetry](./otel) - 分布式追踪
- 🛠️ [CLI 工具](./cli-init) - 代码生成工具

## 常见问题

### Q: 如何修改服务端口？

编辑 `cmd/etc/app.yaml`，修改 `web.server.addr` 配置：

```yaml
web:
  server:
    addr: :9090  # 修改为 9090 端口
```

### Q: 如何启用 HTTPS？

在配置中添加 TLS 证书：

```yaml
web:
  server:
    addr: :8443
    tls:
      cert: /path/to/cert.pem
      key: /path/to/key.pem
```

### Q: 如何在生产环境运行？

将 `development` 设置为 `false`：

```yaml
development: false
```

这将禁用调试模式并启用生产优化。

## 获取帮助

- 📖 查看 [API 文档](https://pkg.go.dev/github.com/tsingsun/woocoo)
- 💬 加入 [Discord 社区](https://discord.gg/358d5uth)
- 🐛 报告 [GitHub Issues](https://github.com/tsingsun/woocoo/issues)
- 💡 查看 [示例项目](https://github.com/tsingsun/woocoo-example)
