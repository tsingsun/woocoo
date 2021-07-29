package redis

import "github.com/go-redis/redis/v8"

type Config struct {
	clusterOpts redis.ClusterOptions
	redisOpts   redis.Options
}
