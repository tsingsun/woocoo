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
	// DocPath is the path to register the playground handler.default is "GET /".
	// If it is empty, the playground will not be registered.
	DocPath string `yaml:"docPath" json:"docPath"`
	// Group is used to be found the matching group router, it must the same as the base path of a route group.
	Group string `yaml:"group" json:"group"`
	// Endpoint is the URL to send GraphQL requests to in the playground.
	Endpoint string `yaml:"endpoint" json:"endpoint"`
	// DocHeader is the header to send GraphQL requests to in the playground.
	DocHeader map[string]string `yaml:"header" json:"header"`
	// WithAuthorization indicates whether parse graphql operations to resource-action data for default authorization.
	//
	// if you want to use custom authorization, you can set it to false, then after RegisterSchema returns the graphql server,
	// Example:
	//
	//   gqlServers,_ := gql.RegisterSchema(router, schema1, schema2)
	//   gqlServers[0].AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {....})
	//
	// Notice: gin.engine.contextWithFallback must be true while using authorization
	WithAuthorization bool `yaml:"withAuthorization" json:"withAuthorization"`
	// AppCode is used to be found the matching app code in the authorization configuration.
	AppCode string `yaml:"appCode" json:"appCode"`
	// use woocoo web handler as graphql OperationHandler Middlewares and ResponseHandler Middlewares
	// Example:
	//
	//  middlewares:
	//    response:
	//      - jwt: ...
	//    operation:
	//      - tenant: ...
	//
	Middlewares   map[string]any `yaml:"middlewares" json:"middlewares"`
	authorization *authz.Authorization
}

var (
	defaultOptions = Options{
		QueryPath:     "/query",
		WebSocketPath: "/query",
		DocPath:       "/",
		Group:         "", // must the same as the base path of a route group
	}
	// optionCache store multiple graphql options, gql-servers can be in different group.
	// TODO in different web server ??
	optionCache = make(map[string]Options)
)

// Handler for graphql
type Handler struct {
	// store multiple graphql options, gql-servers can be in different group
	opts Options
}

// New create a graphql handler for middleware register
func New() *Handler {
	return &Handler{}
}

// RegistryMiddleware register a middleware to web server
func RegistryMiddleware() web.Option {
	return web.WithMiddlewareNewFunc(graphqlHandlerName, Middleware)
}

// Middleware is a builder for initializing graphql handler, it is middleware for gin. see handler.MiddlewareNewFunc
func Middleware() handler.Middleware {
	mw := New()
	return mw
}

func (h *Handler) Name() string {
	return graphqlHandlerName
}

// ApplyFunc is middleware for gin. it seeks the global authorization first, New one if not found.
func (h *Handler) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	var err error
	h.opts = defaultOptions
	if err = cfg.Unmarshal(&h.opts); err != nil {
		panic(err)
	}
	h.opts.authorization = authz.DefaultAuthorization
	if h.opts.WithAuthorization && h.opts.authorization == nil {
		if !cfg.Root().IsSet(authzConfigPath) {
			panic("gql authorization missing authz configuration")
		}
		h.opts.authorization, err = authz.NewAuthorization(cfg.Root().Sub(authzConfigPath))
		if err != nil {
			panic(err)
		}
	}
	if h.opts.Endpoint == "" {
		h.opts.Endpoint = h.opts.Group + h.opts.QueryPath
	}
	optionCache[h.opts.Group] = h.opts
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), gin.ContextKey, c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// RegisterSchema is builder for initializing graphql schemas, initialize order is based on the router group order.
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

// RegisterGraphqlServer is a builder for initializing graphql servers,
// initialize order is based on the router group order.
func RegisterGraphqlServer(websrv *web.Server, servers ...*gqlgen.Server) error {
	index := 0
	for _, gqlserver := range servers {
		if gqlserver == nil {
			continue
		}
		for j := index; j < len(websrv.Router().Groups); j++ {
			gn := websrv.Router().Groups[j].Group.BasePath()
			mid, ok := websrv.HandlerManager().GetMiddleware(handler.GetMiddlewareKey(gn, graphqlHandlerName))
			if ok {
				index = j + 1
				buildGraphqlServer(websrv, websrv.Router().Groups[j], gqlserver, &mid.(*Handler).opts)
				break
			}
		}
	}
	return nil
}

// buildGraphqlServer create a graphiql server
func buildGraphqlServer(websrv *web.Server, routerGroup *web.RouterGroup, server *gqlgen.Server, opt *Options) *gqlgen.Server {
	cnf := conf.NewFromStringMap(opt.Middlewares)
	cnf.Each("operation", func(name string, cfg *conf.Configuration) {
		if hf, ok := websrv.HandlerManager().Get(name); ok {
			mid := hf()
			websrv.HandlerManager().RegisterMiddleware(handler.GetMiddlewareKey(routerGroup.Group.BasePath(), name), mid)
			server.AroundOperations(WrapOperationHandler(mid.ApplyFunc(cfg)))
		}
	})
	cnf.Each("response", func(name string, cfg *conf.Configuration) {
		if hf, ok := websrv.HandlerManager().Get(name); ok {
			mid := hf()
			websrv.HandlerManager().RegisterMiddleware(handler.GetMiddlewareKey(routerGroup.Group.BasePath(), name), mid)
			server.AroundResponses(WrapResponseHandler(mid.ApplyFunc(cfg)))
		}
	})

	if opt.WithAuthorization {
		server.AroundOperations(CheckPermissions(opt))
	}
	var QueryHandler = func(c *gin.Context) {
		server.ServeHTTP(c.Writer, c.Request)
	}

	// set endpoint to graphql-playground used in playground UI
	docHandler := playground.HandlerWithHeaders("graphql", opt.Endpoint, opt.DocHeader)

	var DocHandler = func(c *gin.Context) {
		docHandler.ServeHTTP(c.Writer, c.Request)
	}

	if routerGroup.Group.BasePath() == "/" {
		routerGroup.Router.Engine.POST(opt.QueryPath, QueryHandler)
		if opt.WebSocketPath != "" {
			routerGroup.Router.Engine.GET(opt.WebSocketPath, QueryHandler)
		}
		if opt.DocPath != "" {
			routerGroup.Router.Engine.GET(opt.DocPath, DocHandler)
		}
	} else {
		routerGroup.Group.POST(opt.QueryPath, QueryHandler)
		if opt.WebSocketPath != "" {
			routerGroup.Group.GET(opt.WebSocketPath, QueryHandler)
		}
		if opt.DocPath != "" {
			routerGroup.Group.GET(opt.DocPath, DocHandler)
		}
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

// WrapOperationHandler wrap gin.HandlerFunc to graphql.OperationMiddleware
func WrapOperationHandler(handlerFunc gin.HandlerFunc) graphql.OperationMiddleware {
	return func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		return doOperationHandler(ctx, next, handlerFunc)
	}
}

// WrapResponseHandler wrap gin.HandlerFunc to graphql.ResponseHandler
func WrapResponseHandler(handlerFunc gin.HandlerFunc) graphql.ResponseMiddleware {
	return func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
		return doResponseHandler(ctx, next, handlerFunc)
	}
}

func doResponseHandler(ctx context.Context, next graphql.ResponseHandler, handlerFunc gin.HandlerFunc) *graphql.Response {
	ctx, res := doWebHandler(ctx, handlerFunc)
	if res != nil {
		return res
	}
	return next(ctx)
}

func doWebHandler(ctx context.Context, handlerFunc gin.HandlerFunc) (nctx context.Context, res *graphql.Response) {
	gctx, _ := FromIncomingContext(ctx)
	handler.SetDerivativeContext(gctx, ctx)
	handlerFunc(gctx)
	errList := gqlerror.List{}
	if len(gctx.Errors) > 0 {
		for _, err := range gctx.Errors {
			errList = append(errList, gqlerror.Errorf(err.Error()))
		}
		// if it is a subscription, do not return Response Data
		if gctx.IsWebsocket() {
			gctx.Errors = nil
		}
		res = &graphql.Response{
			Errors: errList,
		}
	}
	nctx = handler.GetDerivativeContext(gctx)
	return
}

func doOperationHandler(ctx context.Context, next graphql.OperationHandler, handlerFunc gin.HandlerFunc) graphql.ResponseHandler {
	ctx, res := doWebHandler(ctx, handlerFunc)
	if res != nil {
		return func(ctx context.Context) *graphql.Response {
			return res
		}
	}
	return next(ctx)
}

// CheckPermissions check the graphql operation permissions base on the package authz
func CheckPermissions(opt *Options) graphql.OperationMiddleware {
	return func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		gctx, _ := FromIncomingContext(ctx)
		op := graphql.GetOperationContext(ctx)
		gp, ok := security.GenericIdentityFromContext(ctx)
		if !ok {
			return func(ctx context.Context) *graphql.Response {
				return &graphql.Response{
					Errors: gqlerror.List{
						gqlerror.Errorf("generic identity not found"),
					},
				}
			}
		}
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
