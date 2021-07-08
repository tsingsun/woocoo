package log

import (
	ejson "encoding/json"
	"github.com/knadh/koanf/parsers/json"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/testdata"
	"go.uber.org/zap"
	"testing"
)

var (
	cnf, _ = conf.BuildWithOption(conf.LocalPath(testdata.Path("app.yaml")))
)

func TestConfig(t *testing.T) {
	var c zap.Config = zap.NewProductionConfig()
	//var data = cnf.Get("log.config")
	p, err := cnf.Operator().Sub("log.config")
	if err != nil {
		t.Error(err)
	}
	bs, _ := p.ToBytes(json.Parser())

	if err := ejson.Unmarshal(bs, &c); err != nil {
		t.Error(err)
	}
	t.Log(c)
}
