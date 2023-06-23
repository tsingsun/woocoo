# WooCoo

[![Language](https://img.shields.io/badge/Language-Go-blue.svg)](https://golang.org/)
[![codecov](https://codecov.io/gh/tsingsun/woocoo/branch/main/graph/badge.svg)](https://codecov.io/gh/tsingsun/woocoo)
[![Go Report Card](https://goreportcard.com/badge/github.com/tsingsun/woocoo)](https://goreportcard.com/report/github.com/tsingsun/woocoo)
[![Build Status](https://github.com/tsingsun/woocoo/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/tsingsun/woocoo/actions?query=branch%3Amain)
[![Release](https://img.shields.io/github/release/tsingsun/woocoo.svg?style=flat-square)](https://github.com/tsingsun/woocoo/releases)
[![GoDoc](https://pkg.go.dev/badge/github.com/tsingsun/woocoo?status.svg)](https://pkg.go.dev/github.com/tsingsun/woocoo?tab=doc)

## Introduction

`WooCoo` is an application development framework and toolkit written in GO(Golang). It is easy to develop WebApi applications or RPC services.

`WooCoo` mainly plays a role of adhesive, and its core components are from other open source projects. 
The current features are as follows:

# Features
- [x] component configurable,easy to split multi environments
- [x] logger and rotate support. [Detail](docs/md/logger.md),
- [x] OpenTelemetry support. [Detail](docs/md/otel.md)
- [x] built-in web router,supports GraphQL.
- [x] built-in grpc server and easy to use grpc client.
- [x] JWT-based validation
- [x] microservice registry and discovery: 
  - etcdv3: register and discovery services 
  - [Polaris](https://github.com/polarismesh/polaris): service discovery and governance

## Tools

- woco-cli: command line tool, include feature:

  - [x] generate code support: `Ent`
  - [x] server code generator for openapi 3.0 spec. see[OpenApi3 Generator](docs/md/oasgen.md)

## Work With

- [facebook ent](https://github.com/ent/ent)
- Graphql: by ent

## examples

[woocoo examples](https://github.com/tsingsun/woocoo-example)

## others:

contact:
- QQ: 21997272

## Thanks

![image](https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.svg)