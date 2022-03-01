# WooCoo

[![Language](https://img.shields.io/badge/Language-Go-blue.svg)](https://golang.org/)
[![codecov](https://codecov.io/gh/tsingsun/woocoo/branch/master/graph/badge.svg)](https://codecov.io/gh/tsingsun/woocoo)
[![Go Report Card](https://goreportcard.com/badge/github.com/tsingsun/woocoo)](https://goreportcard.com/report/github.com/tsingsun/woocoo)
[![Build Status](https://github.com/tsingsun/woocoo/workflows/Run%20Tests/badge.svg?branch=master)](https://github.com/tsingsun/woocoo/actions?query=branch%3Amaster)
[![Release](https://img.shields.io/github/release/tsingsun/woocoo.svg?style=flat-square)](https://github.com/tsingsun/woocoo/releases)
[![GoDoc](https://pkg.go.dev/badge/github.com/tsingsun/woocoo?status.svg)](https://pkg.go.dev/github.com/tsingsun/woocoo?tab=doc)

English | [🇨🇳中文](README_ZH.md)

## Introduction

`WooCoo` is an application development framework and toolkit written in GO(Golang). It is easy to develop WebApi applications or RPC services.

`WooCoo` mainly plays a role of adhesive, and its core components are from other open source projects. 
The current features are as follows:

# Features
- [x] component configurable,easy to split multi environments
- [X] logger and rotate support. [Detail](docs/logger.md),
- [X] OpenTelemetry support. [Detail](docs/otel.md)
- [X] built-in web router,supports GraphQL.
- [X] built-in grpc server.
- [X] JWT-based validation
- [X] microservice registry and discovery: 
  - implements: etcd

# Core Components:

- Logger: [Uber Zap](http://go.uber.org/zap)
- Web: [gin](http://github.com/gin-gonic/gin)

## others:

contact:
- QQ: 21997272

## Thanks

![image](https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.svg)