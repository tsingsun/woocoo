package polarismesh

import (
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/serviceconfig"
	"google.golang.org/grpc/status"
)

var reportInfoAnalyzer ReportInfoAnalyzer = func(info balancer.DoneInfo) (model.RetStatus, uint32) {
	recErr := info.Err
	if nil != recErr {
		st, _ := status.FromError(recErr)
		code := uint32(st.Code())
		return api.RetFail, code
	}
	return api.RetSuccess, 0
}

// SetReportInfoAnalyzer sets report info analyzer
func SetReportInfoAnalyzer(analyzer ReportInfoAnalyzer) {
	reportInfoAnalyzer = analyzer
}

// LBConfig is the LB config for the polaris policy.
type LBConfig struct {
	serviceconfig.LoadBalancingConfig `json:"-"`
	HashKey                           string `json:"hash_key"`
	LbPolicy                          string `json:"lb_policy"`
}
