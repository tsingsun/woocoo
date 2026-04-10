package redisx

import (
	"github.com/redis/go-redis/v9"
	"github.com/tsingsun/woocoo/pkg/conf"
)

// Client is a redis client wrapper. Use redis.UniversalClient instead client/clusterClient since v9
type Client struct {
	redis.UniversalClient
	redisOptions any
}

// NewClient creates a new Redis client from configuration.
//
// The configuration should follow the redis.UniversalOptions format.
// The client automatically adapts to standalone, cluster, or sentinel mode
// based on the provided configuration.
//
// # Configuration Format
//
// ```yaml
// # Standalone mode (single Redis server)
// redis:
//   addrs: ["127.0.0.1:6379"]      # Required: Server address(es)
//   db: 0                          # Optional: Database number (default: 0)
//   password: "your-password"      # Optional: Authentication password
//   username: "your-username"      # Optional: ACL username (Redis 6+)
//
//   # Connection settings
//   dialTimeout: 5s                # Optional: Dial timeout (default: 5s)
//   readTimeout: 3s                # Optional: Read timeout (default: 3s)
//   writeTimeout: 3s               # Optional: Write timeout (default: 3s)
//
//   # Pool settings
//   poolSize: 100                  # Optional: Max active connections (default: 100)
//   minIdleConns: 10               # Optional: Min idle connections (default: 10)
//   poolTimeout: 4s                # Optional: Pool get timeout (default: 4s)
//   connMaxIdleTime: 5m            # Optional: Connection max idle time (default: 5m)
//   connMaxLifetime: 30m           # Optional: Connection max lifetime (default: 30m)
//
//   # Advanced settings
//   protocol: 2                    # Optional: Redis protocol version (2 or 3)
//   clientName: "my-client"        # Optional: Client name for CLIENT SETNAME
//   maxRetries: 3                  # Optional: Max retries on error (default: 3)
//   minRetryBackoff: 8ms           # Optional: Min retry backoff (default: 8ms)
//   maxRetryBackoff: 512ms         # Optional: Max retry backoff (default: 512ms)
// ```
//
// ```yaml
// # Cluster mode (Redis Cluster)
// redis:
//   addrs:
//     - "127.0.0.1:7000"
//     - "127.0.0.1:7001"
//     - "127.0.0.1:7002"
//   password: "cluster-password"
//   readOnly: true                 # Optional: Enable read-only operations on slaves
//   routeByLatency: false          # Optional: Route requests by latency
//   routeRandomly: false           # Optional: Route requests randomly
// ```
//
// ```yaml
// # Sentinel mode (Redis Sentinel for high availability)
// redis:
//   addrs:
//     - "127.0.0.1:26379"
//     - "127.0.0.1:26380"
//   masterName: "mymaster"         # Required: Sentinel master name
//   password: "sentinel-password"  # Password for Redis server
//   sentinelUsername: "sentinel"   # Optional: Sentinel username
//   sentinelPassword: "sentinel-pwd" # Optional: Sentinel password
//   db: 0
// ```
//
// # Examples
//
// ```yaml
// # Example 1: Basic standalone configuration
// store:
//   redis:
//     addrs: ["redis.example.com:6379"]
//     db: 1
//     password: "secret"
// ```
//
// ```yaml
// # Example 2: Cluster configuration with pool settings
// store:
//   redis:
//     addrs:
//       - "cluster-node1:7000"
//       - "cluster-node2:7001"
//       - "cluster-node3:7002"
//     poolSize: 50
//     minIdleConns: 5
//     readTimeout: 5s
// ```
//
// ```yaml
// # Example 3: Sentinel configuration
// store:
//   redis:
//     addrs:
//       - "sentinel1:26379"
//       - "sentinel2:26379"
//       - "sentinel3:26379"
//     masterName: "mymaster"
//     sentinelPassword: "sentinel-secret"
// ```
//
// # Usage in Go Code
//
// ```go
// import "github.com/tsingsun/woocoo/pkg/store/redisx"
//
// // Create client from configuration
// client, err := redisx.NewClient(conf.Global().Sub("store.redis"))
// if err != nil {
//     return err
// }
// defer client.Close()
//
// // Use the client
// err = client.Set(ctx, "key", "value", 0).Err()
// ```
//
// # Notes
//
// - The client mode (standalone/cluster/sentinel) is automatically determined
//   based on the configuration:
//   - If `masterName` is set → Sentinel mode
//   - If multiple `addrs` and no `masterName` → Cluster mode
//   - Otherwise → Standalone mode
// - Time durations can be specified as strings (e.g., "5s", "1m", "1h")
// - All fields are optional except `addrs`
func NewClient(cfg *conf.Configuration) (*Client, error) {
	v := &Client{}
	if err := v.Apply(cfg); err != nil {
		return nil, err
	}
	return v, nil
}

// NewBuiltIn return a Client through application default
func NewBuiltIn() *Client {
	c, err := NewClient(conf.Global().Sub("store.redis"))
	if err != nil {
		panic(err)
	}
	return c
}

func (c *Client) Close() error {
	if c.UniversalClient == nil {
		return nil
	}
	return c.UniversalClient.Close()
}

// Apply implements the conf.Configurable interface
func (c *Client) Apply(cfg *conf.Configuration) error {
	opts := redis.UniversalOptions{}
	err := cfg.Unmarshal(&opts)
	if err != nil {
		return err
	}
	c.redisOptions = &opts
	c.UniversalClient = redis.NewUniversalClient(&opts)
	return nil
}
