package gql

import (
	"context"
	"fmt"
	"github.com/99designs/gqlgen/graphql"
	gqlgen "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/web"
	"github.com/tsingsun/woocoo/web/handler"
	"net/http"
	"runtime/debug"
)

const (
	graphqlHandlerName = "graphql"
)

// Options handler option
type Options struct {
	QueryPath string
	DocPath   string
	// Group must the same as the base path of route group
	Group     string
	SubDomain string
}

var defaultOptions = Options{
	QueryPath: "/query",
	DocPath:   "/",
	Group:     "/graphql", // must the same as the base path of route group
}

type graphqlContextKey struct{}

// Handler for graphql
type Handler struct {
	// store multiple graphql options,gql-servers can be in different group
	opts []Options
}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) Name() string {
	return graphqlHandlerName
}

func (h *Handler) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	opt := defaultOptions
	if err := cfg.Unmarshal(&opt); err != nil {
		panic(err)
	}
	h.opts = append(h.opts, opt)
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), graphqlContextKey{}, c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func (h *Handler) Shutdown() {
}

// RegisterSchema is builder for initializing graphql schemas,initialize order is based on the router group order
func RegisterSchema(websrv *web.Server, schemas ...graphql.ExecutableSchema) (ss []*gqlgen.Server, err error) {
	h, ok := websrv.HandlerManager().Get(graphqlHandlerName)
	if !ok {
		return nil, fmt.Errorf("handler %s not found", graphqlHandlerName)
	}
	for i, schema := range schemas {
		if schema == nil {
			continue
		}
		opt := h.(*Handler).opts[i]
		var rg *web.RouterGroup
		if rg = websrv.Router().FindGroup(opt.Group); rg == nil {
			rg = &web.RouterGroup{Group: &websrv.Router().Engine.RouterGroup, Router: websrv.Router()}
		}
		ss = append(ss, newGraphqlServer(rg, schema, &opt))
	}
	return
}

// newGraphqlServer create a graphiql server
func newGraphqlServer(routerGroup *web.RouterGroup, schema graphql.ExecutableSchema, opt *Options) *gqlgen.Server {
	server := gqlgen.NewDefaultServer(schema)
	server.AroundResponses(func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
		if gctx, err := FromIncomingContext(ctx); err == nil {
			gctx.Set(handler.InnerPath, graphql.GetOperationContext(ctx).OperationName)
		}
		return next(ctx)
	})

	var QueryHandler = func(c *gin.Context) {
		server.ServeHTTP(c.Writer, c.Request)
	}

	var DocHandler = func(c *gin.Context) {
		// set endpoint to graphql-playground used in playground UI
		h := playground.Handler("graphql", opt.SubDomain+opt.Group+opt.QueryPath)
		h.ServeHTTP(c.Writer, c.Request)
	}

	if routerGroup.Group == nil {
		routerGroup.Router.Engine.POST(opt.QueryPath, QueryHandler)
		routerGroup.Router.Engine.GET(opt.DocPath, DocHandler)
	} else {
		routerGroup.Group.POST(opt.QueryPath, QueryHandler)
		routerGroup.Group.GET(opt.DocPath, DocHandler)
	}

	server.SetRecoverFunc(func(ctx context.Context, err interface{}) error {
		gctx, e := FromIncomingContext(ctx)
		if e != nil {
			return e
		}
		handler.HandleRecoverError(gctx, err)
		gctx.AbortWithStatus(http.StatusInternalServerError)
		if conf.Global().Development {
			log.StdPrintln(err)
			debug.PrintStack()
		}
		ue := err.(error)
		return ue
	})

	return server
}

func FromIncomingContext(ctx context.Context) (*gin.Context, error) {
	ginContext := ctx.Value(graphqlContextKey{})
	if ginContext == nil {
		err := fmt.Errorf("could not retrieve gin.Context")
		return nil, err
	}

	gc, ok := ginContext.(*gin.Context)
	if !ok {
		err := fmt.Errorf("gin.Context has wrong type")
		return nil, err
	}
	return gc, nil
}
