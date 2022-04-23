# [WIP] EntImport

ent.io介绍的 [entimport](https://github.com/ariga/entimport) ,但发现如果按照该方式产生的代码,要修改的还是太多,因此从ent底层机制上发,基于模板实现类似的功能

## 快速开始

```
go install github.com/tsingsun/woocoo/cmd/woco
woco entimport --dialect mysql --dsn root:pass@tcp(localhost:3306)/test?parseTime=true --tables user -o ./ent/schema
```

参数说明:
- dialect: 对应数据库类型,目前只支持mysql
- dsn: ConnectionString参数支持环境变量方式,需要注意特殊字符转义
- tables: 指定特定表导出,使用`,`分隔 或者 多`--tables`方式
- o: 输出位置
- qgl: 是否输出graphql文件,位置同-o参数.直接生成relay形式.

## 生成说明

- varchar(45)--> field.String().MaxLen(45)
- int(x): 默认统一生成field.Int().SchemaType(注1),注1,会生成实际的字段类型 

## TODO

- 