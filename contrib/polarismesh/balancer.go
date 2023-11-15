package polarismesh

import (
	"errors"
	"fmt"
	"github.com/polarismesh/polaris-go"
	"google.golang.org/grpc/metadata"
	"strconv"
	"strings"
	"time"

	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
)

var (
	reportInfoAnalyzer ReportInfoAnalyzer = func(info balancer.DoneInfo) (model.RetStatus, uint32) {
		recErr := info.Err
		if nil != recErr {
			st, _ := status.FromError(recErr)
			code := uint32(st.Code())
			return api.RetFail, code
		}
		return api.RetSuccess, 0
	}
	// ErrorPolarisServiceRouteRuleEmpty error service route rule is empty
	ErrorPolarisServiceRouteRuleEmpty = errors.New("service route rule is empty")
)

type (
	// ReportInfoAnalyzer analyze balancer.DoneInfo to polaris report info
	ReportInfoAnalyzer func(info balancer.DoneInfo) (model.RetStatus, uint32)

	balancerBuilder struct {
		name   string
		sdkCtx api.SDKContext
	}

	polarisBalancer struct {
		// the base grpc balancer
		balancer balancer.Balancer

		host    string
		options *dialOptions
		// must init by GetAllInstances, otherwise will be miss cluster info
		response    *model.InstancesResponse
		picker      *polarisNamingPicker
		consumerAPI polaris.ConsumerAPI
		routerAPI   polaris.RouterAPI
		lbCfg       *LBConfig
	}

	pickerBuilder struct {
		balancer *polarisBalancer
	}

	polarisNamingPicker struct {
		balancer *polarisBalancer
		readySCs map[string]balancer.SubConn
		options  *dialOptions
		lbCfg    *LBConfig
		response *model.InstancesResponse
	}
)

// UpdateClientConnState update client connection state. receive from resolver watcher
func (p *polarisBalancer) UpdateClientConnState(state balancer.ClientConnState) error {
	if p.options == nil && state.ResolverState.Attributes != nil {
		p.options = state.ResolverState.Attributes.Value(keyDialOptions).(*dialOptions)
	}
	if state.ResolverState.Attributes != nil {
		resp := state.ResolverState.Attributes.Value(keyResponse).(*model.InstancesResponse)
		if resp.Cluster == nil { // avoid when resolver watcher return single requestInstance return nil cluster info
			resp.Cluster = p.response.Cluster
		}
		p.response = resp
	}
	if state.BalancerConfig != nil {
		p.lbCfg = state.BalancerConfig.(*LBConfig)
	}
	return p.balancer.UpdateClientConnState(state)
}

func (p *polarisBalancer) ResolverError(err error) {
	p.balancer.ResolverError(err)
}

func (p *polarisBalancer) UpdateSubConnState(conn balancer.SubConn, state balancer.SubConnState) {
	p.balancer.UpdateSubConnState(conn, state)
}

func (p *polarisBalancer) Close() {
	p.balancer.Close()
}

// SetReportInfoAnalyzer sets report info analyzer
func SetReportInfoAnalyzer(analyzer ReportInfoAnalyzer) {
	reportInfoAnalyzer = analyzer
}

// NewBalancerBuilder creates a new polaris balancer builder
func NewBalancerBuilder(name string, ctx api.SDKContext) balancer.Builder {
	return &balancerBuilder{
		name:   name,
		sdkCtx: ctx,
	}
}

// Build implements balancer.Builder interface.
func (b balancerBuilder) Build(cc balancer.ClientConn, opts balancer.BuildOptions) balancer.Balancer {
	grpclog.Infof("[Polaris][Balancer] start to build polaris balancer")
	if b.sdkCtx == nil {
		var err error
		b.sdkCtx, err = PolarisContext()
		if err != nil {
			grpclog.Errorln("[Polaris][Balancer] failed to create balancer: " + err.Error())
			return nil
		}
	}
	pb := &pickerBuilder{}
	bb := base.NewBalancerBuilder(b.Name(), pb, base.Config{HealthCheck: true})
	bl := &polarisBalancer{
		balancer:    bb.Build(cc, opts),
		consumerAPI: polaris.NewConsumerAPIByContext(b.sdkCtx),
		routerAPI:   polaris.NewRouterAPIByContext(b.sdkCtx),
	}
	pb.balancer = bl

	return bl
}

func (b balancerBuilder) Name() string {
	return b.name
}

// Build creates a picker, trigger when connection changed to ready
func (pb *pickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	grpclog.Infof("[Polaris][Balancer]: Build called with info: %v", info)
	if len(info.ReadySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}
	scs := make(map[string]balancer.SubConn)
	for sc, subconn := range info.ReadySCs {
		scs[subconn.Address.Addr] = sc
	}
	totalWeight := 0
	readyInstances := make([]model.Instance, 0, len(scs))
	var copyR model.InstancesResponse
	if pb.balancer.response != nil {
		copyR = *pb.balancer.response
		for _, instance := range copyR.Instances {
			// see buildAddressKey
			key := instance.GetHost() + ":" + strconv.FormatInt(int64(instance.GetPort()), 10)
			if _, ok := scs[key]; ok {
				readyInstances = append(readyInstances, instance)
				totalWeight += instance.GetWeight()
			}
		}
		copyR.Instances = readyInstances
		copyR.TotalWeight = totalWeight
	}
	return &polarisNamingPicker{
		balancer: pb.balancer,
		readySCs: scs,
		response: &copyR,
		options:  pb.balancer.options,
	}
}

// Pick an instance from the ready instances
func (pnp *polarisNamingPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	var (
		resp       *model.InstancesResponse
		srcService *model.ServiceInfo
	)
	customRoute := len(pnp.options.SrcMetadata) > 0
	if pnp.options.SrcService != "" {
		srcService = &model.ServiceInfo{
			Service:   pnp.options.SrcService,
			Namespace: getNamespace(pnp.options),
			Metadata:  pnp.options.SrcMetadata,
		}
	}
	if pnp.options.Route {
		request := &polaris.ProcessRoutersRequest{}
		request.DstInstances = pnp.response
		if srcService != nil {
			request.SourceService = *srcService
		}
		if !customRoute {
			if err := pnp.addTrafficLabels(info, request); err != nil {
				grpclog.Errorf("[Polaris][Balancer] fetch traffic labels fail : %+v", err)
			}
		}

		var err error
		resp, err = pnp.balancer.routerAPI.ProcessRouters(request)
		if err != nil {
			return balancer.PickResult{}, err
		}
	} else {
		resp = pnp.response
	}

	lbReq := pnp.buildLoadBalanceRequest(info, resp)
	oneInsResp, err := pnp.balancer.routerAPI.ProcessLoadBalance(lbReq)
	if err != nil {
		return balancer.PickResult{}, err
	}
	targetInstance := oneInsResp.GetInstance()
	addr := fmt.Sprintf("%s:%d", targetInstance.GetHost(), targetInstance.GetPort())
	subSc, ok := pnp.readySCs[addr]
	if ok {
		reporter := &resultReporter{
			instance:      targetInstance,
			consumerAPI:   pnp.balancer.consumerAPI,
			sourceService: srcService,
			startTime:     time.Now(),
		}
		return balancer.PickResult{
			SubConn: subSc,
			Done:    reporter.report,
		}, nil
	}

	return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
}

func (pnp *polarisNamingPicker) buildLoadBalanceRequest(info balancer.PickInfo,
	destIns model.ServiceInstances) *polaris.ProcessLoadBalanceRequest {
	lbReq := &polaris.ProcessLoadBalanceRequest{
		ProcessLoadBalanceRequest: model.ProcessLoadBalanceRequest{
			DstInstances: destIns,
		},
	}
	if pnp.lbCfg != nil {
		if pnp.lbCfg.LbPolicy != "" {
			lbReq.LbPolicy = pnp.lbCfg.LbPolicy
		}
		if pnp.lbCfg.HashKey != "" {
			lbReq.HashKey = []byte(pnp.lbCfg.HashKey)
		}
	}

	// if request scope set Lb Info, use first
	md, ok := metadata.FromOutgoingContext(info.Ctx)
	if ok {
		lbPolicyValues := md.Get(polarisRequestLbPolicy)
		lbHashKeyValues := md.Get(polarisRequestLbHashKey)

		if len(lbPolicyValues) > 0 && len(lbHashKeyValues) > 0 {
			lbReq.LbPolicy = lbPolicyValues[0]
			lbReq.HashKey = []byte(lbHashKeyValues[0])
		}
	}

	return lbReq
}

func (pnp *polarisNamingPicker) addTrafficLabels(info balancer.PickInfo, insReq *polaris.ProcessRoutersRequest) error {
	req := &model.GetServiceRuleRequest{}
	req.Namespace = getNamespace(pnp.options)
	req.Service = pnp.options.Service
	req.SetTimeout(time.Second)
	engine := pnp.balancer.consumerAPI.SDKContext().GetEngine()
	resp, err := engine.SyncGetServiceRule(model.EventRouting, req)
	if err != nil {
		grpclog.Errorf("[Polaris][Balancer] ns:%s svc:%s get route rule fail : %+v",
			req.GetNamespace(), req.GetService(), err)
		return err
	}

	if resp == nil || resp.GetValue() == nil {
		grpclog.Errorf("[Polaris][Balancer] ns:%s svc:%s get route rule empty", req.GetNamespace(), req.GetService())
		return ErrorPolarisServiceRouteRuleEmpty
	}

	routeRule := resp.GetValue().(*apitraffic.Routing)
	labels := collectRouteLabels(routeRule)

	header, ok := metadata.FromOutgoingContext(info.Ctx)
	if !ok {
		header = metadata.MD{}
	}
	for label := range labels {
		if strings.Compare(label, model.LabelKeyPath) == 0 {
			insReq.AddArguments(model.BuildPathArgument(extractBareMethodName(info.FullMethodName)))
			continue
		}
		if strings.HasPrefix(label, model.LabelKeyHeader) {
			values := header.Get(strings.TrimPrefix(label, model.LabelKeyHeader))
			if len(values) > 0 {
				insReq.AddArguments(model.BuildArgumentFromLabel(label, values[0]))
			}
		}
	}

	return nil
}

func collectRouteLabels(routing *apitraffic.Routing) map[string]struct{} {
	ret := make(map[string]struct{})

	for _, rs := range routing.GetInbounds() {
		for _, s := range rs.GetSources() {
			for k := range s.GetMetadata() {
				ret[k] = struct{}{}
			}
		}
	}

	for _, rs := range routing.GetOutbounds() {
		for _, s := range rs.GetSources() {
			for k := range s.GetMetadata() {
				ret[k] = struct{}{}
			}
		}
	}

	return ret
}

type resultReporter struct {
	instance      model.Instance
	consumerAPI   polaris.ConsumerAPI
	startTime     time.Time
	sourceService *model.ServiceInfo
}

// use by balancer.PickResult.Done
func (r *resultReporter) report(info balancer.DoneInfo) {
	if !info.BytesReceived {
		return
	}
	retStatus, code := reportInfoAnalyzer(info)

	callResult := &polaris.ServiceCallResult{}
	callResult.CalledInstance = r.instance
	callResult.RetStatus = retStatus
	callResult.SourceService = r.sourceService
	callResult.SetDelay(time.Since(r.startTime))
	callResult.SetRetCode(int32(code))
	if err := r.consumerAPI.UpdateServiceCallResult(callResult); err != nil {
		grpclog.Errorf("[Polaris][Balancer] report grpc call info fail : %+v", err)
	}
}
