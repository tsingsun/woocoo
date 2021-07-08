package log

import (
	jsonop "encoding/json"
	"fmt"
	"github.com/knadh/koanf/parsers/json"
	"github.com/tsingsun/woocoo/pkg/conf"
	"go.uber.org/zap"
)

const ZapConfigPath = "log.config"

type Config struct {
	zapConfig zap.Config
}

func (c *Config) initConfig(conf *conf.Config) error {
	//zap
	var zc = zap.NewProductionConfig()
	ps, err := conf.Operator().Sub(ZapConfigPath)
	if err != nil {
		return err
	}
	//convert into bytes
	bts, err := ps.ToBytes(json.Parser())
	if err != nil {
		return err
	}

	if err = jsonop.Unmarshal(bts, &zc); err != nil {
		return err
	}
	c.zapConfig = zc
	return nil
}

func (c Config) BuildZap(cnf *conf.Config, opts ...zap.Option) (*zap.Logger, error) {
	if err := c.initConfig(cnf); err != nil {
		return nil, err
	}
	zl, err := c.zapConfig.Build(opts...)

	if err != nil {
		return nil, fmt.Errorf("log component build failed: %s", err)
	}
	return zl, err
}
