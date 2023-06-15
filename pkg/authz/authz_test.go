package authz

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	rediswatcher "github.com/casbin/redis-watcher/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/security"
	"github.com/tsingsun/woocoo/test/testdata"
	"github.com/tsingsun/woocoo/testco/wctest"
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
	// reset
	SetDefaultRequestParserFunc(defaultRequestParser)

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
						"autoSave": true,
						"model":    testdata.Tmp(`rbac_model.conf`),
						"policy":   testdata.Tmp(`rbac_policy.csv`),
					})
				}(),
				opts: []Option{WithRequestParseFunc(func(ctx context.Context, identity security.Identity, item *security.PermissionItem) []any {
					return defaultRequestParser(ctx, identity, item)
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
						"expireTime": 10 * time.Second,
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
				defer got.Watcher.Close()
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
			},
		},
		{
			name: "redis watcher without redis instance",
			args: args{
				cnf: func() *conf.Configuration {
					casbinFilePrepare("rbac")
					return conf.NewFromStringMap(map[string]interface{}{
						"expireTime": 10 * time.Second,
						"watcherOptions": map[string]interface{}{
							"options": map[string]interface{}{
								"addr": "wrong addr",
							},
						},
						"model":  testdata.Tmp(`redis_model.conf`),
						"policy": testdata.Tmp(`redis_policy.csv`),
					})
				}(),
			},
			wantErr: true,
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
		"expireTime": 10 * time.Second,
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
	t.Run("UpdateForAddPolicy", func(t *testing.T) {
		msg := rediswatcher.MSG{ID: uuid.New().String(), Method: "UpdateForAddPolicy",
			Sec: "g", Ptype: "g", NewRule: []string{"alice", "admin"},
		}
		m, err := json.Marshal(msg)
		require.NoError(t, err)
		redis.Publish("/casbin", string(m))
		assert.NoError(t, wctest.RunWait(t, time.Second*3, func() error {
			time.Sleep(time.Second * 2)
			return nil
		}))
	})
	// file adapter does not support UpdateForRemovePolicy
	t.Run("UpdateForRemovePolicy", func(t *testing.T) {
		msg := rediswatcher.MSG{ID: uuid.New().String(), Method: "UpdateForRemovePolicy",
			Sec: "p", Ptype: "p", NewRule: []string{"alice", "data1", "remove"},
		}
		m, err := json.Marshal(msg)
		require.NoError(t, err)
		ok := authz.Enforcer.HasPolicy("alice", "data1", "remove")
		assert.True(t, ok)
		redis.Publish("/casbin", string(m))
		assert.NoError(t, wctest.RunWait(t, time.Second*3, func() error {
			time.Sleep(time.Second * 2)
			return nil
		}))
	})
	authz.Watcher.Close()
}

func TestGetAllowedRecordsForUser(t *testing.T) {
	casbinFilePrepare("conditions")
	authz, err := NewAuthorization(conf.NewFromStringMap(map[string]any{
		"expireTime": 10 * time.Second,
		"model":      testdata.Tmp(`conditions_model.conf`),
		"policy":     testdata.Tmp(`conditions_policy.csv`),
	}))
	require.NoError(t, err)
	condions, err := authz.BaseEnforcer().GetAllowedObjectConditions("alice", "read", "r.obj.")
	require.NoError(t, err)
	assert.Equal(t, []string{"price < 25", "category_id = 2"}, condions)
}
