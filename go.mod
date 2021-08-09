module github.com/tsingsun/woocoo

go 1.15

require (
	github.com/alicebob/miniredis/v2 v2.15.1
	github.com/gin-gonic/gin v1.7.3 // indirect
	github.com/go-redis/cache/v8 v8.4.1
	github.com/go-redis/redis/v8 v8.4.4
	github.com/golang-jwt/jwt v3.2.1+incompatible // indirect
	github.com/knadh/koanf v1.1.1
	github.com/mitchellh/hashstructure/v2 v2.0.2 // indirect
	github.com/mitchellh/mapstructure v1.4.1
	github.com/pelletier/go-toml v1.9.3 // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.4 // indirect
	go.etcd.io/etcd/api/v3 v3.5.0 // indirect
	go.etcd.io/etcd/client/v3 v3.5.0 // indirect
	go.uber.org/atomic v1.8.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.18.1
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/oauth2 v0.0.0-20210628180205-a41e5a781914 // indirect
	golang.org/x/tools v0.1.2 // indirect
	google.golang.org/grpc v1.39.0 // indirect
	google.golang.org/protobuf v1.26.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)

replace gopkg.in/natefinch/lumberjack.v2 => ./third_party/natefinch/lumberjack
