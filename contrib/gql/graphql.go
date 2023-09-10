package gql

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/99designs/gqlgen/graphql"
	gqlgen "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/authz"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/security"
	"github.com/tsingsun/woocoo/web"
	"github.com/tsingsun/woocoo/web/handler"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

const (
	graphqlHandlerName = "graphql"
	authzConfigPath    = "authz"
)

// Options handler option
type Options struct {
	// QueryPath is the path to register the graphql handler.default is "POST /query"
	QueryPath string `yaml:"queryPath" json:"queryPath"`
	// WebSocketPath is the path to register the websocket handler.default is "GET /query"
	WebSocketPath string `yaml:"webSocketPath" json:"webSocketPath"`
	// DocPath is the path to register the playground handler.default is "GET /"
	DocPath string `yaml:"docPath" json:"docPath"`
	// Group is used to be found the matching group router,it must the same as the base path of route group.
	Group string `yaml:"group" json:"group"`
	// Endpoint is the URL to send GraphQL requests to in the playground.
	Endpoint string `yaml:"endpoint" json:"endpoint"`
	// DocHeader is the header to send GraphQL requests to in the playground.
	DocHeader map[string]string `yaml:"header" json:"header"`
	// Skip indicates whether skip the graphql handler.
	Skip bool `yaml:"skip" json:"skip"`
	// WithAuthorization indicates whether parse graphql operations to resource-action data for default authorization.
	//
	// if you want to use custom authorization, you can set it to false, then after RegisterSchema return the graphql server,
	// Example:
	//   gqlServers,_ := gql.RegisterSchema(router, schema1, schema2)
	//   gqlServers[0].AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {....})
	WithAuthorization bool `yaml:"withAuthorization" json:"withAuthorization"`
	// AppCode is used to be found the matching app code in the authorization configuration.
	AppCode string `yaml:"appCode" json:"appCode"`

	authorization *authz.Authorization
}

var defaultOptions = Options{
	QueryPath:     "/query",
	WebSocketPath: "/query",
	DocPath:       "/",
	Group:         "", // must the same as the base path of route group
}

// Handler for graphql
type Handler struct {
	// store multiple graphql options,gql-servers can be in different group
	opts []Options
}

// New create a graphql handler for middleware register
func New() *Handler {
	return &Handler{}
}

func (h *Handler) Name() string {
	return graphqlHandlerName
}

// ApplyFunc is middleware for gin. it seeks the global authorization first, New one if not found.
func (h *Handler) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	var err error
	opt := defaultOptions
	if err = cfg.Unmarshal(&opt); err != nil {
		panic(err)
	}
	opt.authorization = authz.DefaultAuthorization
	if opt.WithAuthorization && opt.authorization == nil {
		if !cfg.Root().IsSet(authzConfigPath) {
			panic("gql authorization missing authz configuration")
		}
		opt.authorization, err = authz.NewAuthorization(cfg.Root().Sub(authzConfigPath))
		if err != nil {
			panic(err)
		}
	}
	h.opts = append(h.opts, opt)
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), gin.ContextKey, c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// Shutdown noting to do
func (h *Handler) Shutdown(_ context.Context) error {
	return nil
}

// RegisterGraphqlServer is builder for initializing graphql servers,initialize order is based on the router group order.
func RegisterGraphqlServer(websrv *web.Server, servers ...*gqlgen.Server) error {
	h, ok := websrv.HandlerManager().Get(graphqlHandlerName)
	if !ok {
		return fmt.Errorf("handler %s not found", graphqlHandlerName)
	}
	for i, gqlserver := range servers {
		if gqlserver == nil {
			continue
		}
		opt := h.(*Handler).opts[i]
		if opt.Skip {
			continue
		}
		if opt.WithAuthorization && !websrv.Router().ContextWithFallback {
			return fmt.Errorf("configuration section 'web.engine.contextWithFallback must be true while using authorization")
		}
		var rg *web.RouterGroup
		if rg = websrv.Router().FindGroup(opt.Group); rg == nil {
			rg = &web.RouterGroup{Group: &websrv.Router().Engine.RouterGroup, Router: websrv.Router()}
		}
		buildGraphqlServer(rg, gqlserver, &opt)
	}
	return nil
}

// RegisterSchema is builder for initializing graphql schemas,initialize order is based on the router group order.
// graphql middleware must registry to web server first though web.RegistryMiddleware(gql.New())
//
// it graphql pkg handler.NewDefaultServer to create a graphql server
func RegisterSchema(websrv *web.Server, schemas ...graphql.ExecutableSchema) (ss []*gqlgen.Server, err error) {
	ss = make([]*gqlgen.Server, len(schemas))
	for i, schema := range schemas {
		ss[i] = gqlgen.NewDefaultServer(schema)
	}
	err = RegisterGraphqlServer(websrv, ss...)
	if err != nil {
		return nil, err
	}
	return ss, err
}

// buildGraphqlServer create a graphiql server
func buildGraphqlServer(routerGroup *web.RouterGroup, server *gqlgen.Server, opt *Options) *gqlgen.Server {
	if opt.WithAuthorization {
		server.AroundOperations(CheckPermissions(opt))
	}
	var QueryHandler = func(c *gin.Context) {
		server.ServeHTTP(c.Writer, c.Request)
	}
	if opt.Endpoint == "" {
		opt.Endpoint = opt.Group + opt.QueryPath
	}
	// set endpoint to graphql-playground used in playground UI
	docHandler := playground.HandlerWithHeaders("graphql", opt.Endpoint, opt.DocHeader)

	var DocHandler = func(c *gin.Context) {
		docHandler.ServeHTTP(c.Writer, c.Request)
	}

	if routerGroup.Group == nil {
		routerGroup.Router.Engine.POST(opt.QueryPath, QueryHandler)
		if opt.WebSocketPath != "" {
			routerGroup.Router.Engine.GET(opt.WebSocketPath, QueryHandler)
		}
		routerGroup.Router.Engine.GET(opt.DocPath, DocHandler)
	} else {
		routerGroup.Group.POST(opt.QueryPath, QueryHandler)
		if opt.WebSocketPath != "" {
			routerGroup.Group.GET(opt.WebSocketPath, QueryHandler)
		}
		routerGroup.Group.GET(opt.DocPath, DocHandler)
	}

	server.SetRecoverFunc(func(ctx context.Context, err any) error {
		gctx, e := FromIncomingContext(ctx)
		if e != nil {
			return e
		}
		handler.HandleRecoverError(gctx, err, 3)
		gctx.AbortWithStatus(http.StatusInternalServerError)
		if conf.Global().Development {
			log.Println(err)
			debug.PrintStack()
		}
		if ue, ok := err.(error); ok {
			return ue
		} else {
			return fmt.Errorf("%v", err)
		}
	})

	return server
}

// FromIncomingContext retrieves the gin.Context from the context.Context
func FromIncomingContext(ctx context.Context) (*gin.Context, error) {
	ginContext, ok := ctx.Value(gin.ContextKey).(*gin.Context)
	if !ok {
		return nil, fmt.Errorf("could not retrieve gin.Context")
	}

	return ginContext, nil
}

// CheckPermissions check the graphql operation permissions base on the package authz
func CheckPermissions(opt *Options) graphql.OperationMiddleware {
	return func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		gctx, _ := FromIncomingContext(ctx)
		// get operation name
		op := graphql.GetOperationContext(ctx)
		gp := security.GenericIdentityFromContext(ctx)
		errList := gqlerror.List{}
		for _, op := range op.Operation.SelectionSet {
			opf := op.(*ast.Field)
			// remove the url path last slash
			pi := &security.PermissionItem{
				AppCode:  opt.AppCode,
				Action:   opf.Name,
				Operator: gctx.Request.Method,
			}
			allowed, err := opt.authorization.CheckPermission(gctx, gp, pi)
			if err != nil {
				errList = append(errList, gqlerror.Errorf("action %s authorization err:%s ", opf.Name, err.Error()))
			}
			if !allowed {
				errList = append(errList, gqlerror.Errorf("action %s not allowed", opf.Name))
			}
		}
		if len(errList) > 0 {
			gctx.Error(errList)
			return func(ctx context.Context) *graphql.Response {
				return &graphql.Response{
					Errors: errList,
				}
			}
		}
		return next(ctx)
	}
}
