根据ent和gqlgen的文档

```
//初始化
go run -mod=mod entgo.io/ent/cmd/ent init
// 配置ent及gqlgen相关文件

generate ./...
//生成时,可能会有额外代码产生,根据实际情况判断是补充代码还是删除
```