---
title: Ent Import
---

## [WIP] EntImport

ent.io介绍的 [entimport](https://github.com/ariga/entimport) ,但发现如果按照该方式产生的代码,要修改的还是太多,因此从ent底层机制上发,基于模板实现类似的功能

## 快速开始

```
go install github.com/tsingsun/woocoo/cmd/woco
woco entimport --dialect mysql --dsn root:pass@tcp(localhost:3306)/test?parseTime=true --tables user -o ./ent/schema
```

参数说明:
- dialect: 对应数据库类型,目前支持mysql,clickhouse
- dsn: ConnectionString参数支持环境变量方式,需要注意特殊字符转义
- tables: 指定特定表导出,使用`,`分隔 或者 多`--tables`方式
- o: 输出位置
- q: 生成relay graphql文件,位置同-o参数.
- i: 是否使用field.Int()映射各int类型字段,如int8,int16等等

## 生成说明

- Nillable or Optional: 如果具有默认值可空,则生成Optional,如果无默认值的可空则两个都生成
- varchar(45)--> field.String().MaxLen(45)
- graphql: 数据库字段同名,排序自动根据索引键产生

## TODO

- 