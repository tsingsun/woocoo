package gql

import (
	"context"
	"encoding/json"
	"github.com/99designs/gqlgen/graphql/handler/testserver"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/contrib/gql/gqltest"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/security"
	"github.com/tsingsun/woocoo/web"
	handler2 "github.com/tsingsun/woocoo/web/handler"
	"net/http"
	"testing"
	"time"
)

const token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxIiwibmFtZSI6InFlZWx5biIsImlhdCI6MTkxNjIzOTAyMn0.x3zlGsLKOjm313FtP9YkXY9IKbtYrEGibjsyPB4X-P8"

func TestSubscription(t *testing.T) {
	cnf := conf.New(conf.WithLocalPath("testdata/app.yaml")).Load()
	hsrv := web.New(
		web.WithConfiguration(cnf.Sub("web2")),
		RegisterMiddleware(),
	)
	hsrv.HandlerManager().Register("user", func() handler2.Middleware {
		return handler2.NewSimpleMiddleware("user", func(cfg *conf.Configuration) gin.HandlerFunc {
			return func(c *gin.Context) {
				p, ok := security.FromContext(c)
				require.True(t, ok)
				require.NotNil(t, p)
			}
		})
	})
	handler := testserver.New()
	handler.AddTransport(transport.Websocket{})

	require.NoError(t, RegisterGraphqlServer(hsrv, handler.Server))

	go func() {
		_ = hsrv.Start(context.Background())
	}()
	defer hsrv.Stop(context.Background())
	time.Sleep(1 * time.Second)
	t.Run("client can receive data", func(t *testing.T) {
		c := gqltest.WsConnect("ws://"+hsrv.ServerOptions().Addr+"/graphql/query", func(h *http.Header) {
			h.Add("Authorization", "Bearer "+token)
		})
		defer c.Close()

		require.NoError(t, c.WriteJSON(&gqltest.OperationMessage{Type: gqltest.ConnectionInitMsg}))
		assert.Equal(t, gqltest.ConnectionAckMsg, gqltest.ReadOp(c).Type)
		assert.Equal(t, gqltest.ConnectionKeepAliveMsg, gqltest.ReadOp(c).Type)

		require.NoError(t, c.WriteJSON(&gqltest.OperationMessage{
			Type:    gqltest.StartMsg,
			ID:      "test_1",
			Payload: json.RawMessage(`{"query": "subscription { name }"}`),
		}))

		handler.SendNextSubscriptionMessage()
		msg := gqltest.ReadOp(c)
		require.Equal(t, gqltest.DataMsg, msg.Type, string(msg.Payload))
		require.Equal(t, "test_1", msg.ID, string(msg.Payload))
		require.Equal(t, `{"data":{"name":"test"}}`, string(msg.Payload))

		handler.SendNextSubscriptionMessage()
		msg = gqltest.ReadOp(c)
		require.Equal(t, gqltest.DataMsg, msg.Type, string(msg.Payload))
		require.Equal(t, "test_1", msg.ID, string(msg.Payload))
		require.Equal(t, `{"data":{"name":"test"}}`, string(msg.Payload))

		require.NoError(t, c.WriteJSON(&gqltest.OperationMessage{Type: gqltest.StopMsg, ID: "test_1"}))

		msg = gqltest.ReadOp(c)
		require.Equal(t, gqltest.CompleteMsg, msg.Type)
		require.Equal(t, "test_1", msg.ID)

		// At this point we should be done and should not receive another message.
		c.SetReadDeadline(time.Now().UTC().Add(1 * time.Millisecond))

		err := c.ReadJSON(&msg)
		if err == nil {
			// This should not send a second close message for the same id.
			require.NotEqual(t, gqltest.CompleteMsg, msg.Type)
			require.NotEqual(t, "test_1", msg.ID)
		} else {
			assert.Contains(t, err.Error(), "timeout")
		}

	})
}
