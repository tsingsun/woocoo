package gql

import (
	"context"
	"fmt"
	"github.com/99designs/gqlgen/graphql"
	gqlgen "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/authz"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/pkg/security"
	"github.com/tsingsun/woocoo/web"
	"github.com/tsingsun/woocoo/web/handler"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
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
	// Group is used to be found the matching group router,it must the same as the base path of route group.
	Group string
	// Endpoint is the URL to send GraphQL requests to in the playground.
	EndPoint string
	Skip     bool
	// WithAuthorization indicates whether parse graphql operations to resource-action data for default authorization.
	//
	// if you want to use custom authorization, you can set it to false, then after RegisterSchema return the graphql server,
	// Example:
	//   gqlServers,_ := gql.RegisterSchema(router, schema1, schema2)
	//   gqlServers[0].AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {....})
	WithAuthorization bool
	// AppCode is used to be found the matching app code in the authorization configuration.
	AppCode       string
	Authorization *authz.Authorization
}

var defaultOptions = Options{
	QueryPath: "/query",
	DocPath:   "/",
	Group:     "/", // must the same as the base path of route group
}

type graphqlContextKey struct{}

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

func (h *Handler) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	var err error
	opt := defaultOptions
	if err = cfg.Unmarshal(&opt); err != nil {
		panic(err)
	}
	opt.Authorization = authz.DefaultAuthorization
	if opt.WithAuthorization && opt.Authorization == nil {
		opt.Authorization, err = authz.NewAuthorization(cfg.Root())
		if err != nil {
			panic(err)
		}
	}
	h.opts = append(h.opts, opt)
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), graphqlContextKey{}, c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// Shutdown noting to do
func (h *Handler) Shutdown(_ context.Context) error {
	return nil
}

// RegisterSchema is builder for initializing graphql schemas,initialize order is based on the router group order.
// graphql middleware must registry to web server first though web.RegistryMiddleware(gql.New())
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
		if opt.Skip {
			continue
		}
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
	if opt.WithAuthorization {
		server.AroundOperations(CheckPermissions(opt))
	}
	var QueryHandler = func(c *gin.Context) {
		server.ServeHTTP(c.Writer, c.Request)
	}
	if opt.EndPoint == "" {
		opt.EndPoint = opt.Group + opt.QueryPath
	}
	// set endpoint to graphql-playground used in playground UI
	docHandler := playground.Handler("graphql", opt.EndPoint)

	var DocHandler = func(c *gin.Context) {
		docHandler.ServeHTTP(c.Writer, c.Request)
	}

	if routerGroup.Group == nil {
		routerGroup.Router.Engine.POST(opt.QueryPath, QueryHandler)
		routerGroup.Router.Engine.GET(opt.DocPath, DocHandler)
	} else {
		routerGroup.Group.POST(opt.QueryPath, QueryHandler)
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
			log.StdPrintln(err)
			debug.PrintStack()
		}
		ue := err.(error)
		return ue
	})

	return server
}

// FromIncomingContext retrieves the gin.Context from the context.Context
func FromIncomingContext(ctx context.Context) (*gin.Context, error) {
	ginContext := ctx.Value(graphqlContextKey{})
	if ginContext == nil {
		return nil, fmt.Errorf("could not retrieve gin.Context")
	}

	return ginContext.(*gin.Context), nil
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
			allowed, err := opt.Authorization.CheckPermission(gctx, gp, pi)
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