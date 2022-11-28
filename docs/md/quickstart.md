---
id: quickstart
---

# 快速开始

## 安装

woocoo需要使用go 1.18以上,相信你已经安装完成了.

```shell
$ mkdir myapp && cd myapp
$ go mod init myapp
$ go get github.com/tsingsun/woocoo

```

## 通过woco cli快速开始

还可以通过cli工具快速创建一个项目来了解:

```
./woco init -p github.com/tsingsun/woocoo/example -m cache,web,grpc,otel -t ./example
cd example
```

以上命令,可以创建web与grpc项目,并且集成了cache,open telemetry支持.

example文件夹的目录结构:

```console
cmd
├──etc
│  └──config.yaml
├──main.go
go.mod
go.sum
README.md
```
此时,已经可以通过`cmd/main.go`运行起一个空逻辑项目.
