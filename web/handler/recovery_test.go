package handler

import (
	"bufio"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/test"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

type ResponseWrite struct {
	httptest.ResponseRecorder
}

func (r ResponseWrite) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	//TODO implement me
	panic("implement me")
}

func (r ResponseWrite) CloseNotify() <-chan bool {
	//TODO implement me
	panic("implement me")
}

func (r ResponseWrite) Status() int {
	return r.Code
}

func (r ResponseWrite) Size() int {
	return 0
}

func (r ResponseWrite) Written() bool {
	return true
}

func (r ResponseWrite) WriteHeaderNow() {
	r.WriteHeader(r.Status())
}

func (r ResponseWrite) Pusher() http.Pusher {
	//TODO implement me
	panic("implement me")
}

func TestHandleRecoverError(t *testing.T) {
	type args struct {
		c   *gin.Context
		err any
	}
	tests := []struct {
		name    string
		args    args
		want    func() any
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "with logger error",
			args: args{
				c: &gin.Context{
					Request: httptest.NewRequest("GET", "/", nil),
					Writer:  &ResponseWrite{},
					Keys: map[string]any{
						AccessLogComponentName: log.NewCarrier(),
					},
				},
				err: errors.New("public error"),
			},
			want: func() any {
				logdata := &test.StringWriteSyncer{}
				log.New(test.NewStringLogger(logdata)).AsGlobal().DisableStacktrace = true
				return logdata
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				ss := i[0].(*test.StringWriteSyncer)
				assert.Len(t, ss.Entry, 0)
				fc := GetLogCarrierFromGinContext(i[1].(*gin.Context))
				assert.NotNil(t, fc)
				assert.Len(t, fc.Fields, 3)
				return true
			},
		},
		{
			name: "without logger",
			args: args{
				c: &gin.Context{
					Request: httptest.NewRequest("GET", "/", nil),
					Writer:  &ResponseWrite{},
				},
				err: "panic",
			},
			want: func() any {
				logdata := &test.StringWriteSyncer{}
				log.New(test.NewStringLogger(logdata)).AsGlobal().DisableStacktrace = true
				return logdata
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				ss := i[0].(*test.StringWriteSyncer)
				all := ss.String()
				assert.Contains(t, all, "request")
				assert.Contains(t, all, "[Recovery from panic]")
				assert.Contains(t, all, "\"component\":\"web\"")
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := tt.want()
			HandleRecoverError(tt.args.c, tt.args.err)
			if !tt.wantErr(t, nil, want, tt.args.c) {
				return
			}
		})
	}
}
