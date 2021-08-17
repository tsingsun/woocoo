package gql

import (
	"context"
	"fmt"
	"github.com/99designs/gqlgen/graphql"
	gqlgen "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web"
	"github.com/tsingsun/woocoo/web/handler"
	"github.com/tsingsun/woocoo/web/handler/logger"
	"github.com/tsingsun/woocoo/web/handler/recovery"
	"net/http"
	"os"
	"runtime/debug"
)

func init() {
	handler.RegisterHandlerFunc("graphql", QraphqlHandler())
}

type graphqlContextKey struct{}

func DefaultGraphqlServer(websrv *web.Server, schema graphql.ExecutableSchema) *gqlgen.Server {
	server := gqlgen.NewDefaultServer(schema)
	server.AroundResponses(func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
		if gctx, err := FromIncomingContext(ctx); err == nil {
			gctx.Set(logger.InnerPath, graphql.GetOperationContext(ctx).OperationName)
		}
		return next(ctx)
	})
	websrv.Router().Engine.POST("/query", func(c *gin.Context) {
		server.ServeHTTP(c.Writer, c.Request)
	})
	websrv.Router().Engine.GET("/", func(c *gin.Context) {
		h := playground.Handler("graphql", "/query")
		h.ServeHTTP(c.Writer, c.Request)
	})
	server.SetRecoverFunc(func(ctx context.Context, err interface{}) error {
		gctx, e := FromIncomingContext(ctx)
		if e != nil {
			return e
		}
		recovery.HandleRecoverError(gctx, err, websrv.Logger(), true)
		gctx.AbortWithStatus(http.StatusInternalServerError)
		if websrv.ServerConfig().Development {
			fmt.Fprintln(os.Stderr, err)
			fmt.Fprintln(os.Stderr)
			debug.PrintStack()
		}
		ue := err.(error)
		return ue
	})

	return server
}

func QraphqlHandler() handler.HandlerApplyFunc {
	return func(handerCfg *conf.Configuration) gin.HandlerFunc {
		return func(c *gin.Context) {
			ctx := context.WithValue(c.Request.Context(), graphqlContextKey{}, c)
			c.Request = c.Request.WithContext(ctx)
			c.Next()
		}
	}
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
