package web

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
)

// RouterGroup is a wrapper for gin.RouterGroup.
type RouterGroup struct {
	Group    *gin.RouterGroup
	basePath string
	Router   *Router
}

// Router is base on Gin
type Router struct {
	*gin.Engine
	Groups        []*RouterGroup
	serverOptions *ServerOptions
}

func NewRouter(options *ServerOptions) *Router {
	gin.SetMode(gin.ReleaseMode)
	gin.DisableConsoleColor()

	r := &Router{
		Engine:        gin.New(),
		serverOptions: options,
	}
	r.Engine.ContextWithFallback = true
	return r
}

func (r *Router) Apply(cnf *conf.Configuration) (err error) {
	if r.serverOptions == nil {
		return errors.New("router apply must apply after Server")
	}
	if err := cnf.Unmarshal(r.Engine); err != nil {
		return err
	}

	cnf.Each("routerGroups", func(group string, sub *conf.Configuration) {
		var mdl []gin.HandlerFunc
		sub.Each("middlewares", func(name string, mid *conf.Configuration) {
			if hf, ok := r.serverOptions.handlerManager.Get(name); ok {
				mdl = append(mdl, hf.ApplyFunc(mid))
			}
		})
		var gr RouterGroup
		// The sequence allows flexible processing of handlers
		gr.basePath = sub.String("basePath")
		gr.Router = r
		if group == "default" {
			if gr.basePath == "" {
				gr.basePath = "/"
			}
			gr.Group = &r.Engine.RouterGroup
			r.Engine.Use(mdl...)
		} else {
			if gr.basePath == "" {
				err = fmt.Errorf("router group: %s must have a basePath", group)
			}
			gr.Group = r.Engine.Group(gr.basePath)
			// clear handlers,let group use self config
			gr.Group.Handlers = gin.HandlersChain{}
			gr.Group.Use(mdl...)
		}
		r.Groups = append(r.Groups, &gr)
	})
	return nil
}

// FindGroup return a specified router group by an url format base path.
//
// parameter basePath is map to configuration:
//
//	routerGroups:
//	- group:
//	  basePath: "/auth"
func (r *Router) FindGroup(basePath string) *RouterGroup {
	if basePath == "" {
		basePath = "/"
	}
	for _, group := range r.Groups {
		if group.basePath == basePath {
			return group
		}
	}
	return nil
}
