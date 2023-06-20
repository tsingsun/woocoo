package authz

import (
	"context"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
	rediswatcher "github.com/casbin/redis-watcher/v2"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/security"
)

var (
	defaultAdapter       persist.Adapter
	DefaultAuthorization *Authorization

	defaultRequestParser = func(_ context.Context, id security.Identity, item *security.PermissionItem) []any {
		return []any{id.Name(), item.Action, item.Operator}
	}
)

type (
	Option func(*Authorization)
	// Authorization is an Authorization feature base on casbin.
	Authorization struct {
		Enforcer     casbin.IEnforcer
		baseEnforcer *casbin.Enforcer
		Watcher      persist.Watcher
		// RequestParser is the function to parse cashbin request according cashbin Model
		RequestParser RequestParserFunc

		autoSave bool
	}
	RequestParserFunc func(ctx context.Context, identity security.Identity, item *security.PermissionItem) []any
)

// WithRequestParseFunc set the request parser function.
func WithRequestParseFunc(f RequestParserFunc) Option {
	return func(authorization *Authorization) {
		authorization.RequestParser = f
	}
}

// NewAuthorization returns a new authenticator with CachedEnforcer and redis watcher by application configuration.
// Configuration example:
//
// authz:
//
//	autoSave: false
//	expireTime: 1h
//	watcherOptions:
//	  options:
//	    addr: "localhost:6379"
//	    channel: "/casbin"
//	model: /path/to/model.conf
//	policy: /path/to/policy.csv
//
// .
// autoSave in watcher callback should be false. but set false will cause casbin main nodes lost save data.
// we will improve in the future.current use database unique index to avoid duplicate data.
func NewAuthorization(cnf *conf.Configuration, opts ...Option) (au *Authorization, err error) {
	au = &Authorization{
		RequestParser: defaultRequestParser,
	}
	for _, opt := range opts {
		opt(au)
	}
	// model
	var dsl, policy any
	m := cnf.String("model")
	if strings.ContainsRune(m, '\n') {
		dsl, err = model.NewModelFromString(m)
		if err != nil {
			return
		}
	} else {
		dsl = cnf.Abs(cnf.String("model"))
	}
	// policy
	if pv := cnf.String("policy"); pv != "" {
		SetAdapter(fileadapter.NewAdapter(pv))
	}
	policy = defaultAdapter
	enforcer, err := casbin.NewCachedEnforcer(dsl, policy)
	if err != nil {
		return
	}

	if cnf.IsSet("expireTime") {
		enforcer.SetExpireTime(cnf.Duration("expireTime"))
	}
	// autosave default to false, because we use redis watcher
	if cnf.IsSet("autoSave") {
		au.autoSave = cnf.Bool("autoSave")
	}
	enforcer.EnableAutoSave(au.autoSave)

	au.Enforcer = enforcer
	au.baseEnforcer = enforcer.Enforcer
	err = au.buildWatcher(cnf)
	if err != nil {
		return
	}

	return
}

func (au *Authorization) buildWatcher(cnf *conf.Configuration) (err error) {
	if !cnf.IsSet("watcherOptions") {
		return
	}
	watcherOptions := rediswatcher.WatcherOptions{
		OptionalUpdateCallback: rediswatcher.DefaultUpdateCallback(au.Enforcer),
	}
	err = cnf.Sub("watcherOptions").Unmarshal(&watcherOptions)
	if err != nil {
		return
	}

	if watcherOptions.Options.Addr != "" {
		au.Watcher, err = rediswatcher.NewWatcher(watcherOptions.Options.Addr, watcherOptions)
	} else if watcherOptions.ClusterOptions.Addrs != nil {
		au.Watcher, err = rediswatcher.NewWatcherWithCluster(watcherOptions.Options.Addr, watcherOptions)
	}
	if err != nil {
		return
	}
	return au.Enforcer.SetWatcher(au.Watcher)
}

func (au *Authorization) CheckPermission(ctx context.Context, identity security.Identity, item *security.PermissionItem) (bool, error) {
	return au.Enforcer.Enforce(au.RequestParser(ctx, identity, item)...)
}

// BaseEnforcer returns the base enforcer. casbin api is not broadcasting to enforcer interface. so need to use base enforcer.
func (au *Authorization) BaseEnforcer() *casbin.Enforcer {
	return au.baseEnforcer
}

// SetAdapter sets the default adapter for the enforcer.
func SetAdapter(adapter persist.Adapter) {
	defaultAdapter = adapter
}

// SetDefaultAuthorization sets the default authorization.
func SetDefaultAuthorization(au *Authorization) {
	DefaultAuthorization = au
}

// SetDefaultRequestParserFunc sets the default request parser function.
//
// notice: in web middleware that will implicit initial authorization component,
// web permission check use appcode,c.Request.URL.Path, request.Method as default enforce check,
// you should set this first if not default.
func SetDefaultRequestParserFunc(f RequestParserFunc) {
	defaultRequestParser = f
}
