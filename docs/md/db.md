---
id: db
---

# 数据库

## 数据库连接

可基于配置初始化`sql.db`实例.

```yaml
dbname:
  # 数据库驱动名称
  driverName: mysql
  # 数据库连接字符串
  dsn: root:${password}@tcp(127.0.0.1:3306)
  # 最大空闲连接数
  maxIdleConns: 10
  # 最大打开连接数
  maxOpenConns: 100
  # 连接最大存活时间
  connMaxLifetime:
  # 数据库密码加密,如不需要加密可去掉本节点
  encryption:
    # 加密后的数据库密码
    password: U2FsdGVkX1+tlVEqk7q5J4HmwH0tZg
    # 数据库加密方式,目前只支持aes-gcm加密, 因此可不需要配置
    method: aes-gcm
```

如果基于安全需求,为了不在配置文件中体现密码明文,配置支持对数据库密码加密, 此时在dsn需要使用`${password}`做为占位符.同时指定环境变量`DB_SECRET_KEY`为AES-GCM加密的密钥.

```go
cfg := conf.New()
db := sqlx.NewSqlDB(cfg.Sub("dbname"))
```