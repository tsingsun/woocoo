package web

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web/handler"
	"reflect"
	"runtime"
)

type RuleManager map[string]gin.RouteInfo

var registerRule = RuleManager{}
var registerHandler = map[string]gin.HandlerFunc{}

//Router is base on Gin
//you can use AfterRegisterInternalHandler to replace an inline HandlerFunc or add a new
type Router struct {
	Engine                       *gin.Engine
	server                       *Server
	hms                          RuleManager
	AfterRegisterInternalHandler func(*Router)
}

func NewRouter(s *Server) *Router {
	if s.config != nil && !s.config.Development {
		gin.SetMode(gin.ReleaseMode)
		gin.DisableConsoleColor()
	}
	return &Router{
		Engine: gin.New(),
		server: s,
		hms:    RuleManager{},
	}
}

func (r *Router) Apply(cfg *conf.Configuration, path string) error {
	if r.server == nil {
		return errors.New("router apply must apply after Server")
	}
	if err := cfg.Parser().UnmarshalByJson(path, r.Engine); err != nil {
		return err
	}

	registerInternalHandler(r)
	//hfs := cfg.Sub(path + conf.KeyDelimiter + "handleFuncs").ParserOperator().MapKeys()
	hfs := cfg.Sub(path).ParserOperator().Slices("handleFuncs")
	if r.AfterRegisterInternalHandler != nil {
		r.AfterRegisterInternalHandler(r)
	}
	for _, k := range hfs {
		var name string
		for s, _ := range k.Raw() {
			name = s
			break
		}
		if hf, ok := registerHandler[name]; ok {
			r.Engine.Use(hf)
		} else {
			return errors.New("middleware not found:" + name)
		}
	}
	return nil
}

//Collect convert RoutesInfo into map,
//Note: Gin.Engine.Routes() return the last rule for each rule
func (r *Router) Collect() RuleManager {
	rs := r.Engine.Routes()
	for _, info := range rs {
		if _, ok := r.hms[info.Handler]; !ok {
			r.hms[info.Path] = info
		}
	}
	return r.hms
}

//RehandleRule use for adding customize route rules into Router
func (r *Router) RehandleRule() error {
	for _, ruleInfo := range registerRule {
		r.Engine.Handle(ruleInfo.Method, ruleInfo.Path, ruleInfo.HandlerFunc)
	}
	return nil
}

//RegisterRouteRule Support register a route rule with only one HandlerFunc
func RegisterRouteRule(path, method string, handlerFunc gin.HandlerFunc) error {
	var ri = gin.RouteInfo{
		Path:        path,
		Method:      method,
		HandlerFunc: handlerFunc,
		Handler:     runtime.FuncForPC(reflect.ValueOf(handlerFunc).Pointer()).Name(),
	}
	registerRule[ri.Path] = ri
	return nil
}

func RegisterHandlerFunc(name string, handlerFunc gin.HandlerFunc) {
	//if _, ok := registerHandler[name]; ok {
	//	panic("handlerFunc exists:" + name)
	//}
	registerHandler[name] = handlerFunc
}

func registerInternalHandler(router *Router) {
	RegisterHandlerFunc("accessLog", handler.AccessLogHandler(router.server.logger))
	RegisterHandlerFunc("recovery", handler.RecoveryHandler(router.server.logger, true))
	RegisterHandlerFunc("auth", handler.AuthHandler(router.server.configuration))
}
