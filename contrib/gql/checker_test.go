package gql

import (
	"testing"

	gqlgen "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/testserver"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/assert"
)

func TestSupportStream(t *testing.T) {
	t.Run("not subs schema", func(t *testing.T) {
		srv := gqlgen.New(&gqlSchemaMock)
		assert.False(t, SupportStream(srv))
	})
	t.Run("empty transport", func(t *testing.T) {
		srv := testserver.New()
		assert.False(t, SupportStream(srv.Server))
	})
	t.Run("sse", func(t *testing.T) {
		srv := testserver.New()
		srv.AddTransport(transport.SSE{})
		assert.True(t, SupportStream(srv.Server))
	})
	t.Run("mix-with-sse", func(t *testing.T) {
		srv := testserver.New()
		srv.AddTransport(transport.POST{})
		srv.AddTransport(transport.SSE{})
		assert.True(t, SupportStream(srv.Server))
	})
}
