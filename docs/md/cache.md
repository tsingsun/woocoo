---
id: cache
---
# 缓存

`cache`包提供一套多级缓存机制,具备以下特点:

1. 支持二级缓存: 本地缓存与redis缓存配合,结合Getter有效的补充数据, 可在运行时指定缓存值级别.
2. 防止缓存击穿: 采用singleflight机制.
3. 防止缓存穿透: 配合本地缓存后,对于不存在的数据,会在本地缓存中设置一个空值,防止缓存穿透.
4. 配置化,可扩展,易用性高.

缓存组件为可扩展组件,缓存使用接口如下:

```go
// Cache is the interface for cache.
type Cache interface {
	// Get gets the value from cache and unmarshal it to v.
	Get(ctx context.Context, key string, value any, opts ...Option) error
	// Set sets the value to cache.
	Set(ctx context.Context, key string, value any, opts ...Option) error
	// Has reports whether value for the given key exists.
	Has(ctx context.Context, key string) bool
	// Del deletes the value for the given key.
	Del(ctx context.Context, key string) error
	// IsNotFound detect the error weather not found from cache
	IsNotFound(err error) bool
}

// 初始化组件时注册,后使用..
_ = redisc.New(conf.Global()).Register()
var value string
cache.Get(context.background(),"key",&value)
cache.Set(context.background(),"key",value,cache.WithTTL(10*time.Second))
// 通过getter在无缓存时可通过Getter补充数据,一般Getter函数为数据库查询等,因此采用防止缓存穿透的功能
cache.Get(context.background(),"key",&value,cache.WithGetter(func() (any, error) {
    return "value",nil // DB query
}))
```

插件实现了缓存接口后.注册到组件中,即可使用.

Option方法有:

- WithTTL: 设置缓存过期时间,在Get时,如果配合内存缓存,则会从远程缓存自动更新缓存本地,同时设置本地缓存的过期时间会更加精确. 
- WithGroup: 采用singleflight,防止缓存击穿,在同一时间,只有一个请求会去获取数据,其他请求会等待,直到获取到数据.
- WithGetter: 在Get时,如果缓存不存在,则会调用Getter获取数据,并且设置到缓存中,强制采用WithGroup.
- WithSkip: 指定要访问的缓存级别.
  - SkipLocal: 忽略本地缓存处理.
  - SkipRemote: 忽略远程缓存处理.
  - SkipCache: 忽略本地与远程缓存,如果有设置Getter则执行.
- WithRaw: 内存缓存是否采用原始值,

> 以上Option的支持情况取决于插件的实现.内置的Redis插件都支持.

## 内存缓存

内存缓存可独立使用,也做为其他远程缓存的二级缓存使用,配置及说明见Redis缓存介绍

内存缓存无需Group设置,特别说明注意:

WithRaw指示了即是否进行序列化处理.需要自行注意数据安全.默认为false.对远程或跨进程缓存是无效的,因此必然会序列化.
简单的说值类型及引用类型之差,在确认数据为线程安全情况下,可以设置为true,以提高性能.不然一般不建议设置为true.

内存缓存的TTL当做为二级缓存时是可额外配置,考虑到二级缓存的作用为只是为了短时间的缓存,因此不建议设置过长的时间.如以下场景:

1. 防止缓存击穿: 由于空值也会被存储,可缓解该问题, 
2. 提高整体缓存性能.

## Redis缓存

缓存采用的内存缓存与Redis的组合缓存. 内存缓存采用的是[tinylru](https://github.com/vmihailenco/go-tinylfu)算法与配合Redis.

1. 可以设置单独启用其中一种.
2. 在两种缓存都启用的情况下,会先从本地LFU缓存中获取,如果没有命中,则从Redis缓存中获取,并且将获取到的数据写入本地LFU缓存中.

### 配置

```yaml
# redis缓存名称,可配置多个但是名称不能重复
driverName: redis
# 内存缓存配置
local:
  # 内存缓存容量,必须指定 > 1 
  size: 100000
  # 过期时间,默认1分钟,如果Set方法未指定,则采用此过期时间
  ttl: 10m
  # 内置的小型布隆过滤器的容量,默认100000
  samples: 100000
# 以下为redis option配置,同store redis配置,可查询go-redis文档: 
# 如果指定了 masterName 选项，则返回 FailoverClient 哨兵客户端。
# 如果 Addrs 是2个以上的地址，则返回 ClusterClient 集群客户端。
# 其他情况，返回 Client 单节点客户端。
addrs:
  - 127.0.0.1:6379
db: 0
```

附: [go-redis配置文档](https://redis.uptrace.dev/zh/guide/go-redis-option.html)