package authz

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/testdata"
	"os"
	"testing"
)

func casbinFilePrepare(node string) {
	p, err := conf.NewParserFromFile(testdata.Path("authz/casbin.yaml"))
	if err != nil {
		panic(err)
	}
	cfg := conf.NewFromParse(p)
	if err := os.WriteFile(testdata.Tmp(node+`_policy.csv`), []byte(cfg.String(node+".policy")), os.ModePerm); err != nil {
		panic(err)
	}
	if err := os.WriteFile(testdata.Tmp(node+`_model.conf`), []byte(cfg.String(node+".model")), os.ModePerm); err != nil {
		panic(err)
	}
}

func TestNewAuthorization(t *testing.T) {
	type args struct {
		cnf *conf.Configuration
	}
	tests := []struct {
		name  string
		args  args
		panic bool
		check func(t *testing.T, got *Authorization)
	}{
		{
			name: "RBAC",
			args: args{
				cnf: func() *conf.Configuration {
					casbinFilePrepare("rbac")
					return conf.NewFromStringMap(map[string]interface{}{
						"model":  testdata.Tmp(`rbac_model.conf`),
						"policy": testdata.Tmp(`rbac_policy.csv`),
					})
				}(),
			},
			panic: false,
			check: func(t *testing.T, got *Authorization) {
				assert.NoError(t, got.Enforcer.LoadPolicy())
				_, err := got.Enforcer.AddPermissionForUser("alice", "data1", "write")
				require.NoError(t, err)
				has, err := got.Enforcer.Enforce("alice", "data1", "write")
				require.NoError(t, err)
				assert.True(t, has)
				assert.NoError(t, got.Enforcer.SavePolicy())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var do = func() {
				got := NewAuthorization(tt.args.cnf)
				tt.check(t, got)
			}
			if tt.panic {
				assert.NotPanics(t, func() {
					do()
				})
			} else {
				do()
			}
		})
	}
}
