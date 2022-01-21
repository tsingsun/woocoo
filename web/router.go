package web

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web/handler"
	"github.com/tsingsun/woocoo/web/handler/auth"
	"github.com/tsingsun/woocoo/web/handler/logger"
	"github.com/tsingsun/woocoo/web/handler/recovery"
	"reflect"
	"runtime"
)

type RuleManager map[string]gin.RouteInfo

var registerRule = RuleManager{}

//Router is base on Gin
//you can use AfterRegisterInternalHandler to replace an inline HandlerFunc or add a new
type Router struct {
	Engine                       *gin.Engine
	Groups                       []*gin.RouterGroup
	server                       *Server
	hms                          RuleManager
	AfterRegisterInternalHandler func(*Router)
}

func NewRouter(s *Server) *Router {
	if !s.serverSetting.Development {
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
	if err := cfg.Parser().Unmarshal(path, r.Engine); err != nil {
		return err
	}

	registerInternalHandler(r)
	//rgs := cfg.Sub(path + conf.KeyDelimiter + "handleFuncs").ParserOperator().MapKeys()
	rgs := cfg.Sub(path).ParserOperator().Slices("routerGroups")
	if r.AfterRegisterInternalHandler != nil {
		r.AfterRegisterInternalHandler(r)
	}
	for _, rItem := range rgs {
		var name string
		for s := range rItem.Raw() {
			name = s
			break
		}
		var gr *gin.RouterGroup
		rCfg := rItem.Cut(name)
		// The sequence allows flexible processing of handlers
		if name == "default" {
			gr = &r.Engine.RouterGroup
		} else {
			gr = r.Engine.Group(rCfg.String("basePath"))
			gr.Handlers = gin.HandlersChain{}
		}
		r.Groups = append(r.Groups, gr)
		hfs := rCfg.Slices("handleFuncs")
		for _, hItem := range hfs {
			var fname string
			for s := range hItem.Raw() {
				fname = s
				break
			}
			if hf, ok := handler.RegisterHandler[fname]; ok {
				subCfg := cfg.CutFromOperator(hItem.Cut(fname))
				gr.Use(hf(subCfg))
			} else {
				return errors.New("middleware not found:" + fname)
			}
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

func (r Router) GetGroup(basePath string) *gin.RouterGroup {
	for _, group := range r.Groups {
		if group.BasePath() == basePath {
			return group
		}
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

func registerInternalHandler(router *Router) {
	handler.RegisterHandlerFunc("accessLog", logger.AccessLogHandler(router.server.logger))
	handler.RegisterHandlerFunc("recovery", recovery.RecoveryHandler(router.server.logger, true))
	handler.RegisterHandlerFunc("auth", auth.AuthHandler())
}
