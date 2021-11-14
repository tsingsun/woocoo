package log

import (
	"fmt"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/third_party/natefinch/lumberjack"
	"go.uber.org/zap"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
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

func (c *Config) initConfig(cfg *conf.Configuration) error {
	//zap
	var zc = zap.NewProductionConfig()
	if err := cfg.Parser().Unmarshal(zapConfigPath, &zc); err != nil {
		return err
	}
	if cfg.IsSet("development") {
		zc.Development = cfg.Bool("development")
	}
	c.zapConfig = &zc

	var otps []string
	for _, path := range c.zapConfig.OutputPaths {
		u, err := convertPath(path, cfg.GetBaseDir(), cfg.IsSet(rotateConfigPath))
		if err != nil {
			return err
		}
		otps = append(otps, u)
	}
	c.zapConfig.OutputPaths = otps

	if cfg.IsSet(rotateConfigPath) {
		l := &lumberjack.Logger{ConcurrentSafe: false}
		if err := cfg.Parser().Unmarshal(rotateConfigPath, l); err != nil {
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
			if runtime.GOOS == "windows" {
				l.Filename = strings.TrimPrefix(u.Path, "/")
			} else {
				l.Filename = u.Path
			}
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

func (c *Config) BuildZap(cnf *conf.Configuration, opts ...zap.Option) (*zap.Logger, error) {
	if err := c.initConfig(cnf); err != nil {
		return nil, err
	}
	zl, err := c.zapConfig.Build(opts...)

	if err != nil {
		return nil, fmt.Errorf("log component build failed: %s", err)
	}
	return zl, err
}

func (c *Config) buildZap(opts ...zap.Option) {

}

func convertPath(path string, base string, useRotate bool) (string, error) {
	u, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("can't parse %q as a URL: %v", path, err)
	}
	if path == "stdout" || path == "stderr" || (u.Scheme != "" && u.Scheme != "file") {
		return path, nil
	}
	if u.Scheme != "file" {
		var absPath string
		if !filepath.IsAbs(u.Path) {
			absPath = filepath.Join(base, path)
			if runtime.GOOS == "windows" {
				absPath = "/" + absPath
			}
			u.Path = absPath
		}
	}
	if useRotate {
		u.Scheme = rotateSchema
	}
	return u.String(), nil
}
