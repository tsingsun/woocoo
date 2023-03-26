package authz

import (
	"context"
	"encoding/json"
	"github.com/alicebob/miniredis/v2"
	rediswatcher "github.com/casbin/redis-watcher/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/internal/wctest"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/security"
	"github.com/tsingsun/woocoo/test/testdata"
	"os"
	"testing"
	"time"
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
		cnf  *conf.Configuration
		opts []Option
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		check   func(t *testing.T, got *Authorization)
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
				opts: []Option{WithRequestParseFunc(func(ctx context.Context, identity security.Identity, item *security.PermissionItem) []any {
					return defaultRequestParserFunc(ctx, identity, item)
				})},
			},
			wantErr: false,
			check: func(t *testing.T, got *Authorization) {
				assert.NotNil(t, got.RequestParser)
				assert.NoError(t, got.Enforcer.LoadPolicy())
				_, err := got.Enforcer.AddPermissionForUser("alice", "data1", "write")
				require.NoError(t, err)
				has, err := got.Enforcer.Enforce("alice", "data1", "write")
				require.NoError(t, err)
				assert.True(t, has)
				assert.NoError(t, got.Enforcer.SavePolicy())
			},
		},
		{
			name: "redis watcher",
			args: args{
				cnf: func() *conf.Configuration {
					casbinFilePrepare("redis")
					mr := miniredis.RunT(t)
					return conf.NewFromStringMap(map[string]interface{}{
						"watcherOptions": map[string]interface{}{
							"options": map[string]interface{}{
								"addr": mr.Addr(),
							},
						},
						"model":  testdata.Tmp(`redis_model.conf`),
						"policy": testdata.Tmp(`redis_policy.csv`),
					})
				}(),
			},
			wantErr: false,
			check: func(t *testing.T, got *Authorization) {
				gotcallback := make(chan bool)
				require.NoError(t, got.Watcher.SetUpdateCallback(func(s string) {
					gotcallback <- true
				}))
				assert.NoError(t, got.Enforcer.LoadPolicy())
				_, err := got.Enforcer.AddPermissionForUser("alice", "data1", "write")
				require.NoError(t, err)
				has, err := got.Enforcer.Enforce("alice", "data1", "write")
				require.NoError(t, err)
				assert.True(t, has)
				has, err = got.CheckPermission(context.Background(), security.NewGenericPrincipalByClaims(jwt.MapClaims{
					"sub": "alice",
				}).Identity(), &security.PermissionItem{
					Action:   "data1",
					Operator: "write",
				})
				require.NoError(t, err)
				assert.True(t, has)
				assert.NoError(t, got.Enforcer.SavePolicy())
				for {
					select {
					case <-gotcallback:
						return
					case <-time.After(time.Second * 2):
						t.Error("timeout")
						return
					}
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewAuthorization(tt.args.cnf, tt.args.opts...)
			SetDefaultAuthorization(got)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			tt.check(t, got)
		})
	}
}

func TestRedisCallback(t *testing.T) {
	casbinFilePrepare("callback")
	redis := miniredis.RunT(t)
	authz, err := NewAuthorization(conf.NewFromStringMap(map[string]any{
		"watcherOptions": map[string]any{
			"options": map[string]any{
				"addr":    redis.Addr(),
				"channel": "/casbin",
			},
		},
		"model":  testdata.Tmp(`callback_model.conf`),
		"policy": testdata.Tmp(`callback_policy.csv`),
	}))
	require.NoError(t, err)
	t.Parallel()
	t.Run("UpdateForAddPolicy", func(t *testing.T) {
		//authz.Enforcer.AddRoleForUser("alice", "admin")
		//authz.Enforcer.SavePolicy()
		msg := rediswatcher.MSG{ID: uuid.New().String(), Method: "UpdateForAddPolicy",
			Sec: "g", Ptype: "g", NewRule: []string{"alice", "admin"},
		}
		m, err := json.Marshal(msg)
		require.NoError(t, err)
		redis.Publish("/casbin", string(m))
		err = wctest.RunWait(t, time.Second*2, func() error {
			time.Sleep(time.Second)
			ok, err := authz.Enforcer.HasRoleForUser("alice", "admin")
			assert.NoError(t, err)
			assert.True(t, ok)
			return nil
		})
		assert.NoError(t, err)
	})
	// file watcher does not support UpdateForRemovePolicy
	t.Run("UpdateForRemovePolicy", func(t *testing.T) {
		msg := rediswatcher.MSG{ID: uuid.New().String(), Method: "UpdateForRemovePolicy",
			Sec: "p", Ptype: "p", NewRules: [][]string{{"alice", "data1", "remove"}},
		}
		m, err := json.Marshal(msg)
		require.NoError(t, err)
		ok := authz.Enforcer.HasPolicy("alice", "data1", "remove")
		assert.True(t, ok)
		redis.Publish("/casbin", string(m))
		err = wctest.RunWait(t, time.Second*2, func() error {
			time.Sleep(time.Second)
			ok := authz.Enforcer.HasPolicy("alice", "data1", "remove")
			assert.True(t, ok)
			return nil
		})
		assert.NoError(t, err)
	})
}
