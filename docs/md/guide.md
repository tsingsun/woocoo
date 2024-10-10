---
id: guide
---

# 简介

WooCoo(武库)的定位是一个基于Golang的应用开发框架及工具包,核心组件选取了优秀的开源项目,而WooCoo希望做一个优秀的粘合剂,将这些优秀的组件集成起来,追求优秀的开发体验与工程实践.

## web

提供了以WebAPI为开发目的的Web服务.

通过[快速开始](quickstart)来了解WooCoo.

### grpc

提供了GRPC的微服务体系.

## woocoo cli

woocoo cli工具: 名为 **woco** 代码减化,意为减轻开发人员工作的代码生成工具,让繁锁的工作交给工具完成.

## Benchmark

- woocoo web对gin的组件优化体现出了正效果.
- grpc: TODO

### Web

- goos: darwin
- goarch: amd64
- cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz

| Name                 | (1)    | (2)         | (3)        | (4)          |
|----------------------|--------|-------------|------------|--------------|
| WooCooWebDefault     | 564612 | 2633 ns/op  | 1103 B/op	 | 5 allocs/op  |
|                      |        |             |            |              |
| GinDefault           | 81198  | 14418 ns/op | 354 B/op   | 13 allocs/op |
| GinDefaultMockLogger | 423054 | 2747 ns/op  | 221 B/op   | 8 allocs/op  |

> gin default使用了自带了低性能的stdout logger 所以我们使用了一个内存的MockLogger做测试.而woocoo web是使用stdout做为输出的.

- (1) Total Repetitions achieved in constant time, higher means more confident result
- (2) Single Repetition Duration (ns/op), lower is better
- (3) Heap Memory (B/op), lower is better
- (4) Average Allocations per Repetition (allocs/op), lower is better
