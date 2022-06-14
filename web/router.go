package web

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
)

// RouterGroup is a wrapper for echo.Group.
// echo.Group is too sample and does not support search.
type RouterGroup struct {
	Group    *gin.RouterGroup
	basePath string
	Router   *Router
}

// Router is base on Gin
// you can use AfterRegisterInternalHandler to replace an inline HandlerFunc or add a new
type Router struct {
	*gin.Engine
	Groups                       []*RouterGroup
	serverOptions                *ServerOptions
	AfterRegisterInternalHandler func(*Router)
}

func NewRouter(options *ServerOptions) *Router {
	gin.SetMode(gin.ReleaseMode)
	gin.DisableConsoleColor()

	r := &Router{
		Engine:        gin.New(),
		serverOptions: options,
	}
	return r
}

func (r *Router) Apply(cfg *conf.Configuration) error {
	if r.serverOptions == nil {
		return errors.New("router apply must apply after Server")
	}
	if err := cfg.Unmarshal(r.Engine); err != nil {
		return err
	}

	rgs := cfg.ParserOperator().Slices("routerGroups")
	if r.AfterRegisterInternalHandler != nil {
		r.AfterRegisterInternalHandler(r)
	}
	for _, rItem := range rgs {
		var name string
		for s := range rItem.Raw() {
			name = s // key is the router group name
			break
		}

		rCfg := rItem.Cut(name)
		hfs := rCfg.Slices("middlewares")

		var mdl []gin.HandlerFunc
		for _, hItem := range hfs {
			var fname string
			for s := range hItem.Raw() {
				fname = s
				break
			}
			if hf, ok := r.serverOptions.handlerManager.Get(fname); ok {
				subhf := hItem.Cut(fname)
				// if subhf is empty,pass the original config
				if len(subhf.Keys()) == 0 {
					subhf = hItem
				}
				mdl = append(mdl, hf.ApplyFunc(cfg.CutFromOperator(subhf)))
			} else {
				return errors.New("middleware not found:" + fname)
			}
		}
		var gr RouterGroup
		// The sequence allows flexible processing of handlers
		gr.basePath = rCfg.String("basePath")
		gr.Router = r
		if name == "default" {
			if gr.basePath == "" {
				gr.basePath = "/"
			}
			r.Engine.Use(mdl...)
		} else {
			if gr.basePath == "" {
				return fmt.Errorf("router group: %s must have a basePath", name)
			}
			gr.Group = r.Engine.Group(gr.basePath)
			// clear handlers,let group use self config
			gr.Group.Handlers = gin.HandlersChain{}
			gr.Group.Use(mdl...)
		}
		r.Groups = append(r.Groups, &gr)
	}
	return nil
}

// FindGroup return a specified router group by an url format base path.
//
// parameter basePath is map to configuration:
//   routerGroups:
//     - group:
//         basePath: "/auth"
func (r *Router) FindGroup(basePath string) *RouterGroup {
	for _, group := range r.Groups {
		if group.basePath == basePath {
			return group
		}
	}
	return nil
}
