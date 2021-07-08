module github.com/tsingsun/woocoo

go 1.15

require (
	github.com/knadh/koanf v1.1.1
	github.com/mitchellh/mapstructure v1.4.1
	github.com/pelletier/go-toml v1.9.3 // indirect
	go.uber.org/atomic v1.8.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.18.1
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/tools v0.1.2 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace (
	gopkg.in/natefinch/lumberjack.v2 => ./third_party/natefinch/lumberjack
)