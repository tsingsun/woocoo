package gql

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/99designs/gqlgen/graphql/handler/testserver"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/contrib/gql/gqltest"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/security"
	"github.com/tsingsun/woocoo/web"
	"github.com/tsingsun/woocoo/web/handler"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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
	hsrv.HandlerManager().Register("user", func() handler.Middleware {
		return handler.NewSimpleMiddleware("user", func(cfg *conf.Configuration) gin.HandlerFunc {
			return func(c *gin.Context) {
				p, ok := security.FromContext(c)
				require.True(t, ok)
				require.NotNil(t, p)
			}
		})
	})
	testServer := testserver.New()
	testServer.AddTransport(transport.Websocket{})

	require.NoError(t, RegisterGraphqlServer(hsrv, testServer.Server))

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

		testServer.SendNextSubscriptionMessage()
		msg := gqltest.ReadOp(c)
		require.Equal(t, gqltest.DataMsg, msg.Type, string(msg.Payload))
		require.Equal(t, "test_1", msg.ID, string(msg.Payload))
		require.Equal(t, `{"data":{"name":"test"}}`, string(msg.Payload))

		testServer.SendNextSubscriptionMessage()
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

func TestSSE(t *testing.T) {
	cnfStr := `
web:
  server:
    addr: 127.0.0.1:0
  engine:
    routerGroups:
    - graphql:
        basePath: "/graphql"
        middlewares:
        - graphql:
`
	cnf := conf.NewFromBytes([]byte(cnfStr))

	initializeWithServer := func() (*testserver.TestServer, *web.Server) {
		hsrv := web.New(
			web.WithConfiguration(cnf.Sub("web")),
			RegisterMiddleware(),
		)

		testServer := testserver.New()
		testServer.AddTransport(transport.SSE{})
		require.NoError(t, RegisterGraphqlServer(hsrv, testServer.Server))
		return testServer, hsrv
	}

	createHTTPTestRequest := func(query string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/graphql/query", strings.NewReader(query))
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("content-type", "application/json; charset=utf-8")
		return req
	}

	createHTTPRequest := func(url string, query string) *http.Request {
		req, err := http.NewRequest("POST", url, strings.NewReader(query))
		require.NoError(t, err, "Request threw error -> %s", err)
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("content-type", "application/json; charset=utf-8")
		return req
	}

	readLine := func(br *bufio.Reader) string {
		bs, err := br.ReadString('\n')
		require.NoError(t, err)
		return bs
	}

	t.Run("stream failure", func(t *testing.T) {
		_, h := initializeWithServer()
		req := httptest.NewRequest(http.MethodPost, "/graphql/query", strings.NewReader(`{"query":"subscription { name }"}`))
		req.Header.Set("content-type", "application/json; charset=utf-8")
		w := httptest.NewRecorder()
		h.Router().ServeHTTP(w, req)
		assert.Equal(t, 400, w.Code, "Request return wrong status -> %d", w.Code)
		assert.Equal(t, `{"errors":[{"message":"transport not supported"}],"data":null}`, w.Body.String())
	})

	t.Run("decode failure", func(t *testing.T) {
		_, h := initializeWithServer()
		req := createHTTPTestRequest("notjson")
		w := httptest.NewRecorder()
		h.Router().ServeHTTP(w, req)
		assert.Equal(t, 400, w.Code, "Request return wrong status -> %d", w.Code)
		assert.Equal(t, `{"errors":[{"message":"json request body could not be decoded: invalid character 'o' in literal null (expecting 'u') body:notjson"}],"data":null}`, w.Body.String())
	})

	t.Run("parse failure", func(t *testing.T) {
		_, h := initializeWithServer()
		req := createHTTPTestRequest(`{"query":"subscription {{ name }"}`)
		w := httptest.NewRecorder()
		h.Router().ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code, "Request return wrong status -> %d", w.Code)
		assert.Equal(t, "keep-alive", w.Header().Get("Connection"))
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))

		br := bufio.NewReader(w.Body)

		assert.Equal(t, ":\n", readLine(br))
		assert.Equal(t, "\n", readLine(br))
		assert.Equal(t, "event: next\n", readLine(br))
		assert.Equal(t, "data: {\"errors\":[{\"message\":\"Expected Name, found {\",\"locations\":[{\"line\":1,\"column\":15}],\"extensions\":{\"code\":\"GRAPHQL_PARSE_FAILED\"}}],\"data\":null}\n", readLine(br))
		assert.Equal(t, "\n", readLine(br))
		assert.Equal(t, "event: complete\n", readLine(br))
		assert.Equal(t, "\n", readLine(br))

		_, err := br.ReadByte()
		assert.Equal(t, err, io.EOF)
	})

	t.Run("subscribe", func(t *testing.T) {
		h, srv := initializeWithServer()
		go func() {
			if err := srv.Start(context.Background()); err != nil {
				panic(err)
			}
		}()
		defer srv.Stop(context.Background())
		time.Sleep(1 * time.Second)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.SendNextSubscriptionMessage()
		}()

		client := &http.Client{}
		url := fmt.Sprintf("http://%s/graphql/query", srv.ServerOptions().Addr)
		req := createHTTPRequest(url, `{"query":"subscription { name }"}`)
		res, err := client.Do(req)
		require.NoError(t, err, "Request threw error -> %s", err)
		defer func() {
			require.NoError(t, res.Body.Close())
		}()

		assert.Equal(t, 200, res.StatusCode, "Request return wrong status -> %d", res.Status)
		assert.Equal(t, "keep-alive", res.Header.Get("Connection"))
		assert.Equal(t, "text/event-stream", res.Header.Get("Content-Type"))

		br := bufio.NewReader(res.Body)

		assert.Equal(t, ":\n", readLine(br))
		assert.Equal(t, "\n", readLine(br))
		assert.Equal(t, "event: next\n", readLine(br))
		assert.Equal(t, "data: {\"data\":{\"name\":\"test\"}}\n", readLine(br))
		assert.Equal(t, "\n", readLine(br))

		wg.Add(1)
		go func() {
			defer wg.Done()
			h.SendNextSubscriptionMessage()
		}()

		assert.Equal(t, "event: next\n", readLine(br))
		assert.Equal(t, "data: {\"data\":{\"name\":\"test\"}}\n", readLine(br))
		assert.Equal(t, "\n", readLine(br))

		wg.Add(1)
		go func() {
			defer wg.Done()
			h.SendCompleteSubscriptionMessage()
		}()

		assert.Equal(t, "event: complete\n", readLine(br))
		assert.Equal(t, "\n", readLine(br))

		_, err = br.ReadByte()
		assert.Equal(t, err, io.EOF)

		wg.Wait()
	})
}
