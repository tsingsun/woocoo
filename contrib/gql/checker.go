package gql

import (
	gqlgen "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"net/http"
	"strings"
)

type testResponseWriter struct {
	http.ResponseWriter
	header http.Header
	status int
	body   []byte
}

func (t *testResponseWriter) Header() http.Header {
	return t.header
}

func (t *testResponseWriter) Write(bytes []byte) (int, error) {
	t.body = bytes
	return len(bytes), nil
}

func (t *testResponseWriter) WriteHeader(statusCode int) {
	t.status = statusCode
}

// SupportStream checks whether the server supports streaming
func SupportStream(server *gqlgen.Server) bool {
	var checkSupportedResponse = func(w *testResponseWriter) bool {
		return !(w.status == http.StatusBadRequest && strings.Contains(string(w.body), "transport not supported"))
	}
	// Websocket request, make it error thought miss Connection header
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		panic(err)
	}
	r.Header.Set("Upgrade", "websocket")
	ts := &transport.Websocket{}
	if !ts.Supports(r) {
		panic("testStreamSupport invalid, please check graphql package")
	}
	w := &testResponseWriter{
		header: make(http.Header),
	}
	server.ServeHTTP(w, r)
	if checkSupportedResponse(w) {
		return true
	}
	// sse
	r, err = http.NewRequest(http.MethodPost, "/", nil)
	if err != nil {
		panic(err)
	}
	r.Header.Set("Accept", "text/event-stream")
	r.Header.Set("Content-Type", "application/json")
	w = &testResponseWriter{
		header: make(http.Header),
	}
	sse := &transport.SSE{}
	if !sse.Supports(r) {
		panic("testStreamSupport invalid, please check code")
	}
	server.ServeHTTP(w, r)
	if checkSupportedResponse(w) {
		return true
	}
	return false
}
