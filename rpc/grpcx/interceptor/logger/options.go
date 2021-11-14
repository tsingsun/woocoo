package logger

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"time"
)

var defautlOptions = &options{
	TimestampFormat: time.RFC3339,
}

type options struct {
	TimestampFormat string `json:"TimestampFormat" yaml:"TimestampFormat"`
}

func (o *options) Apply(cfg *conf.Configuration) {
	if err := cfg.Parser().Unmarshal("", o); err != nil {
		panic(err)
	}
}
