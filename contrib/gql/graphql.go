package gql

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	gqlgen "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/security"
	"github.com/tsingsun/woocoo/web"
	"github.com/tsingsun/woocoo/web/handler"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"go.uber.org/zap"
)

const (
	graphqlHandlerName = "graphql"
	useStream          = "sse_or_websocket"
)

var (
	ErrMissGinContext = errors.New("could not retrieve gin.Context")
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
	DocHeader map[string]string `yaml:"docHeader" json:"docHeader"`
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
	Middlewares map[string]any `yaml:"middlewares" json:"middlewares"`

	devMode bool
	// whether support websocket or sse.
	isSupportStream bool
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

// Handler for graphql.
//
// While Subscription, response flow is defference from query or mutation.
type Handler struct {
	// store multiple graphql options, gql-servers can be in different group
	opts Options
}

// New create a graphql handler for middleware register
func New() *Handler {
	return &Handler{}
}

// RegisterMiddleware register a middleware to web server
func RegisterMiddleware() web.Option {
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
	h.opts.devMode = cfg.Development
	if h.opts.WithAuthorization && security.DefaultAuthorizer == nil {
		panic("security.DefaultAuthorizer is nil")
	}

	if h.opts.Endpoint == "" {
		h.opts.Endpoint = h.opts.Group + h.opts.QueryPath
	}
	optionCache[h.opts.Group] = h.opts
	return func(c *gin.Context) {
		if h.opts.isSupportStream && CheckStreamConnection(c) {
			c.Set(useStream, true)
		}
		ctx := context.WithValue(c.Request.Context(), gin.ContextKey, c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// RegisterSchema is builder for initializing graphql schemas, initialize order is based on the router group order.
// graphql middleware must registry to web server first though web.RegisterMiddleware(gql.New())
//
// you must not add transport for websocket or sse after call this method that will cause Options.isSupportStream not correct.
// if you want to use websocket or sse, you must call RegisterGraphqlServer.
func RegisterSchema(websrv *web.Server, schemas ...graphql.ExecutableSchema) (ss []*gqlgen.Server, err error) {
	ss = make([]*gqlgen.Server, len(schemas))
	for i, schema := range schemas {
		srv := gqlgen.New(schema)
		srv.AddTransport(transport.Options{})
		srv.AddTransport(transport.GET{})
		srv.AddTransport(transport.POST{})
		srv.AddTransport(transport.MultipartForm{})
		srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
		ss[i] = srv
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
			mid, ok := websrv.HandlerManager().GetMiddleware(web.GetMiddlewareKey(gn, graphqlHandlerName))
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
func buildGraphqlServer(websrv *web.Server, routerGroup *web.RouterGroup, server *gqlgen.Server, opt *Options) {
	opt.isSupportStream = SupportStream(server)
	cnf := conf.NewFromStringMap(opt.Middlewares)
	cnf.Each("operation", func(name string, cfg *conf.Configuration) {
		if hf, ok := websrv.HandlerManager().Get(name); ok {
			mid := hf()
			websrv.HandlerManager().RegisterMiddleware(web.GetMiddlewareKey(routerGroup.Group.BasePath(), name), mid)
			server.AroundOperations(WrapOperationHandler(mid.ApplyFunc(cfg)))
		}
	})
	cnf.Each("response", func(name string, cfg *conf.Configuration) {
		switch name {
		case handler.AccessLogName:
			server.AroundResponses(newStreamLogger().ApplyFunc(cfg))
		default:
			if hf, ok := websrv.HandlerManager().Get(name); ok {
				mid := hf()
				websrv.HandlerManager().RegisterMiddleware(web.GetMiddlewareKey(routerGroup.Group.BasePath(), name), mid)
				server.AroundResponses(WrapResponseHandler(mid.ApplyFunc(cfg)))
			}
		}
	})

	if opt.WithAuthorization {
		server.AroundOperations(CheckPermissions(opt))
	}
	var QueryHandler = func(c *gin.Context) {
		server.ServeHTTP(c.Writer, c.Request)
	}

	// set endpoint to graphql-playground used in playground UI
	docHandler := playground.HandlerWithHeaders("graphql", opt.Endpoint, opt.DocHeader, opt.DocHeader)

	var DocHandler = func(c *gin.Context) {
		docHandler.ServeHTTP(c.Writer, c.Request)
	}
	var addHandler = func(gr interface {
		POST(string, ...gin.HandlerFunc) gin.IRoutes
		GET(string, ...gin.HandlerFunc) gin.IRoutes
	}) {
		gr.POST(opt.QueryPath, QueryHandler)
		if opt.WebSocketPath != "" {
			gr.GET(opt.WebSocketPath, QueryHandler)
		}
		if opt.DocPath != "" {
			server.Use(extension.Introspection{})
			gr.GET(opt.DocPath, DocHandler)
		}
	}
	if routerGroup.Group.BasePath() == "/" {
		addHandler(routerGroup.Router.Engine)
	} else {
		addHandler(routerGroup.Group)
	}

	server.SetRecoverFunc(func(ctx context.Context, err any) error {
		if opt.devMode {
			log.Println(err)
			debug.PrintStack()
		}
		gctx, e := FromIncomingContext(ctx)
		if e != nil {
			return fmt.Errorf("gql.SetRecoverFunc: %v : %v", err, e)
		}
		handler.HandleRecoverError(gctx, err, 3)
		// clear errors, let gql server handle those.
		clear(gctx.Errors)
		if ue, ok := err.(error); ok {
			return ue
		} else {
			return fmt.Errorf("%v", err)
		}
	})
	// Work with handler.ErrorHandler and replace graphql.DefaultErrorPresenter(ctx, err)
	server.SetErrorPresenter(ErrorPresenter)
}

// ErrorPresenter is a graphql.ErrorPresenterFunc. it tries to convert gin.Error to gqlerror.Error and mask the error by woocoo handler.ErrorHandler
// and keep the original error to log in access logger.
func ErrorPresenter(ctx context.Context, err error) *gqlerror.Error {
	gctx, _ := FromIncomingContext(ctx)
	if gctx != nil {
		if fc := handler.GetLogCarrierFromGinContext(gctx); fc != nil {
			fc.Fields = append(fc.Fields, zap.Error(err))
		}
	}
	var gqlErr *gqlerror.Error
	if !errors.As(err, &gqlErr) {
		gqlErr = &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Err:     err,
			Message: err.Error(),
		}
	}
	var ginErr *gin.Error
	if errors.As(err, &ginErr) {
		code, errTxt := handler.LookupErrorCode(uint64(ginErr.Type), ginErr.Err)
		if code > 0 {
			gqlErr.Err = errors.New(errTxt)
			gqlErr.Message = errTxt
		} else {
			code = int(ginErr.Type)
		}
		if gqlErr.Extensions == nil {
			gqlErr.Extensions = make(map[string]any)
		}
		gqlErr.Extensions["code"] = code
		if ginErr.Meta != nil {
			gqlErr.Extensions["meta"] = ginErr.Meta
		}
	}
	return gqlErr
}

// FromIncomingContext retrieves the gin.Context from the context.Context
func FromIncomingContext(ctx context.Context) (*gin.Context, error) {
	ginContext, ok := ctx.Value(gin.ContextKey).(*gin.Context)
	if !ok {
		return nil, ErrMissGinContext
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
	c, err := FromIncomingContext(ctx)
	if err != nil {
		return next(ctx)
	}
	ctx, res := doWebHandler(ctx, c, handlerFunc)
	if res != nil {
		return res
	}
	return next(ctx)
}

func doWebHandler(ctx context.Context, c *gin.Context, handlerFunc gin.HandlerFunc) (context.Context, *graphql.Response) {
	handler.SetDerivativeContext(c, ctx)
	handlerFunc(c)
	ctx = handler.GetDerivativeContext(c)
	errList := gqlerror.List{}

	var res *graphql.Response
	if len(c.Errors) > 0 {
		for _, err := range c.Errors {
			errList = append(errList, ErrorPresenter(ctx, err))
		}
		// if it is a subscription, do not return Response Data
		if isStreamConnection(c) {
			c.Errors = nil
		}
		res = &graphql.Response{
			Errors: errList,
		}
	}
	return ctx, res
}

func doOperationHandler(ctx context.Context, next graphql.OperationHandler, handlerFunc gin.HandlerFunc) graphql.ResponseHandler {
	c, err := FromIncomingContext(ctx)
	if err != nil {
		return next(ctx)
	}
	ctx, res := doWebHandler(ctx, c, handlerFunc)
	if res != nil {
		if isStreamConnection(c) {
			return graphql.OneShot(res)
		} else {
			return func(ctx context.Context) *graphql.Response {
				return res
			}
		}
	}
	return next(ctx)
}

// CheckPermissions check the graphql operation permissions base on the package authz
func CheckPermissions(opt *Options) graphql.OperationMiddleware {
	return func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		gctx, err := FromIncomingContext(ctx)
		if err != nil {
			return envResponseError(gctx, gqlerror.List{gqlerror.Errorf("%s", err)})
		}
		op := graphql.GetOperationContext(ctx)
		errList := gqlerror.List{}
		for _, op := range op.Operation.SelectionSet {
			opf := op.(*ast.Field)
			// introspectSchema or introspectType ignore
			if opf.Name == "__type" || opf.Name == "__schema" {
				continue
			}
			allowed, err := security.IsAllowed(gctx, security.ArnKindGql, opt.AppCode, gctx.Request.Method, opf.Name)
			if err != nil {
				errList = append(errList, gqlerror.Errorf("action %s is not allowed due to err:%s ", opf.Name, err))
			} else if !allowed {
				errList = append(errList, gqlerror.Errorf("action %s is not allowed", opf.Name))
			}
		}
		if len(errList) > 0 {
			gctx.AbortWithStatus(http.StatusForbidden)
			return envResponseError(gctx, errList)
		}
		return next(ctx)
	}
}

func envResponseError(c *gin.Context, errList gqlerror.List) graphql.ResponseHandler {
	res := &graphql.Response{
		Errors: errList,
	}
	if isStreamConnection(c) {
		c.Errors = nil
		return graphql.OneShot(res)
	}
	return func(ctx context.Context) *graphql.Response {
		return res
	}
}

func isStreamConnection(c *gin.Context) bool {
	_, ok := c.Get(useStream)
	return ok
}

// CheckStreamConnection check if the request is a stream connection which is a websocket or SSE request.
func CheckStreamConnection(c *gin.Context) bool {
	if c.IsWebsocket() {
		return true
	}
	if strings.Contains(strings.ToLower(c.Request.Header.Get("Accept")), "text/event-stream") {
		return true
	}
	return false
}
