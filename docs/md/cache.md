# 缓存

缓存采用的本地LFU与Redis的缓存组合.

1. 并且可以设置单独启用其中一种.
2. 在两种缓存都启用的情况下,会先从本地LFU缓存中获取,如果没有命中,则从Redis缓存中获取,并且将获取到的数据写入本地LFU缓存中.

## 本地LFU缓存

本地LFU缓存采用的是[github.com/golang/groupcache/lru](
