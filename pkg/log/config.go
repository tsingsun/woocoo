package log

import (
	"fmt"
	"github.com/tsingsun/woocoo/pkg/conf"
	"go.uber.org/zap"
	"gopkg.in/natefinch/lumberjack.v2"
	"net/url"
	"path/filepath"
)

const (
	zapConfigPath    = "log.config"
	rotateConfigPath = "log.rotate"
	// rotate:[//[userinfo@]host][/]path[?query][#fragment]
	rotateSchema = "rotate"
)

type Config struct {
	zapConfig *zap.Config
	isRolling bool
	rotate    *lumberjack.Logger
}

func (c *Config) initConfig(conf *conf.Config) error {
	//zap
	var zc = zap.NewProductionConfig()
	if err := conf.Operator().UnmarshalByJson(zapConfigPath, &zc); err != nil {
		return err
	}

	c.zapConfig = &zc

	if conf.IsSet(rotateConfigPath) {
		var otps []string
		for _, path := range c.zapConfig.OutputPaths {
			u, err := convertPath(path)
			if err != nil {
				return err
			}
			otps = append(otps, u)
		}
		c.zapConfig.OutputPaths = otps
		l := &lumberjack.Logger{ConcurrentSafe: false}
		if err := conf.Operator().UnmarshalByJson(rotateConfigPath, l); err != nil {
			return err
		}

		err := zap.RegisterSink(rotateSchema, func(u *url.URL) (zap.Sink, error) {
			if u.User != nil {
				return nil, fmt.Errorf("user and password not allowed with file URLs: got %v", u)
			}
			if u.Fragment != "" {
				return nil, fmt.Errorf("fragments not allowed with file URLs: got %v", u)
			}
			if u.RawQuery != "" {
				return nil, fmt.Errorf("query parameters not allowed with file URLs: got %v", u)
			}
			// Error messages are better if we check hostname and port separately.
			if u.Port() != "" {
				return nil, fmt.Errorf("ports not allowed with file URLs: got %v", u)
			}
			if hn := u.Hostname(); hn != "" && hn != "localhost" {
				return nil, fmt.Errorf("file URLs must leave host empty or use localhost: got %v", u)
			}
			l.Filename = u.Path
			return l, nil
		})
		if err != nil {
			return err
		}

		c.isRolling = true
		c.rotate = l
	}
	return nil
}

func (c *Config) BuildZap(cnf *conf.Config, opts ...zap.Option) (*zap.Logger, error) {
	if err := c.initConfig(cnf); err != nil {
		return nil, err
	}
	zl, err := c.zapConfig.Build(opts...)

	if err != nil {
		return nil, fmt.Errorf("log component build failed: %s", err)
	}
	return zl, err
}

func convertPath(path string) (string, error) {
	u, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("can't parse %q as a URL: %v", path, err)
	}
	if path == "stdout" || path == "stderr" || (u.Scheme != "" && u.Scheme != "file") {
		return path, nil
	}
	u.Scheme = rotateSchema
	if !filepath.IsAbs(u.Path) {
		u.Path = filepath.Join(conf.BaseDir, u.Path)
	}
	return u.String(), nil
}
