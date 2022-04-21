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
	"github.com/tsingsun/woocoo/web/handler/logger"
	"github.com/tsingsun/woocoo/web/handler/recovery"
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
	h, ok := websrv.HandlerManager().GetHandler(graphqlHandlerName)
	if !ok {
		return nil, fmt.Errorf("handler %s not found", graphqlHandlerName)
	}
	for i, schema := range schemas {
		if schema == nil {
			continue
		}
		opt := h.(*Handler).opts[i]
		var rg *gin.RouterGroup
		if rg = websrv.Router().Group(opt.Group); rg == nil {
			rg = &websrv.Router().Engine.RouterGroup
		}
		ss = append(ss, newGraphqlServer(rg, schema, &opt))
	}
	return
}

// newGraphqlServer create a graphiql server
func newGraphqlServer(routerGroup *gin.RouterGroup, schema graphql.ExecutableSchema, opt *Options) *gqlgen.Server {
	server := gqlgen.NewDefaultServer(schema)
	server.AroundResponses(func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
		if gctx, err := FromIncomingContext(ctx); err == nil {
			gctx.Set(logger.InnerPath, graphql.GetOperationContext(ctx).OperationName)
		}
		return next(ctx)
	})

	routerGroup.POST(opt.QueryPath, func(c *gin.Context) {
		server.ServeHTTP(c.Writer, c.Request)
	})
	routerGroup.GET(opt.DocPath, func(c *gin.Context) {
		// set endpoint to graphql-playground used in playground UI
		h := playground.Handler("graphql", opt.SubDomain+opt.Group+opt.QueryPath)
		h.ServeHTTP(c.Writer, c.Request)
	})

	server.SetRecoverFunc(func(ctx context.Context, err interface{}) error {
		gctx, e := FromIncomingContext(ctx)
		if e != nil {
			return e
		}
		recovery.HandleRecoverError(gctx, err, log.Component(log.WebComponentName), true)
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
