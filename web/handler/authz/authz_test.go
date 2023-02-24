package authz

import (
	"github.com/tsingsun/woocoo/web/handler"
	"testing"
)

func TestNewAuthorizer(t *testing.T) {
	tests := []struct {
		name string
		want *Authorizer
	}{
		{
			name: "NewAuthorizer",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New()
			handler.NewSimpleMiddleware("authz", got.ApplyFunc)
		})
	}
}
