# WooCoo

[![Language](https://img.shields.io/badge/Language-Go-blue.svg)](https://golang.org/)
[![codecov](https://codecov.io/gh/tsingsun/woocoo/branch/master/graph/badge.svg)](https://codecov.io/gh/tsingsun/woocoo)
[![Go Report Card](https://goreportcard.com/badge/github.com/tsingsun/woocoo)](https://goreportcard.com/report/github.com/tsingsun/woocoo)
[![Build Status](https://github.com/tsingsun/woocoo/actions/workflows/gotest.yml/badge.svg?branch=master)](https://github.com/tsingsun/woocoo/actions?query=branch%3Amaster)
[![Release](https://img.shields.io/github/release/tsingsun/woocoo.svg?style=flat-square)](https://github.com/tsingsun/woocoo/releases)
[![GoDoc](https://pkg.go.dev/badge/github.com/tsingsun/woocoo?status.svg)](https://pkg.go.dev/github.com/tsingsun/woocoo?tab=doc)

English | [🇨🇳中文](docs/README_zh.md)

## Introduction

`WooCoo` is an application development framework and toolkit written in GO(Golang). It is easy to develop WebApi applications or RPC services.

`WooCoo` mainly plays a role of adhesive, and its core components are from other open source projects. 
The current features are as follows:

# Features
- [x] component configurable,easy to split multi environments
- [X] logger and rotate support. [Detail](docs/logger.md),
- [X] OpenTelemetry support. [Detail](docs/otel.md)
- [X] built-in web router,supports GraphQL.
- [X] built-in grpc server and easy to use grpc client.
- [X] JWT-based validation
- [X] microservice registry and discovery: 
  - etcdv3: register and discovery services 
  - [Polaris](https://github.com/polarismesh/polaris): service discovery and governance

## Core Components:

- Logger: [Uber Zap](http://go.uber.org/zap)
- Web: [gin](http://github.com/gin-gonic/gin)

## Tools

- woco-cli: command line tool. see [Detail](./cmd/woco/README.md)

  generate code support: `Ent`

## Work With

- [facebook ent](https://github.com/ent/ent)
- Graphql: by ent

## others:

contact:
- QQ: 21997272

## Thanks

![image](https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.svg)