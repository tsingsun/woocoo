package authz

import (
	"context"
	"errors"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
	rediswatcher "github.com/casbin/redis-watcher/v2"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/pkg/security"
	"strings"
)

var (
	defaultAdapter persist.Adapter

	DefaultAuthorization *Authorization
)

type (
	Option func(*Authorization)
	// Authorization is an Authorization feature base on casbin.
	Authorization struct {
		Enforcer casbin.IEnforcer
		Watcher  persist.Watcher
		// RequestParser is the function to parse cashbin request according cashbin Model
		RequestParser RequestParserFunc
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
//		authz:
//	   expireTime: 1h
//		  watcherOptions:
//		    options:
//		      addr: "localhost:6379"
//		      channel: "/casbin"
//		  model: /path/to/model.conf
//		  policy: /path/to/policy.csv
func NewAuthorization(cnf *conf.Configuration, opts ...Option) (au *Authorization, err error) {
	au = &Authorization{
		RequestParser: defaultRequestParserFunc,
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
		dsl = cnf.String("model")
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

	au.Enforcer = enforcer
	au.Enforcer.EnableAutoSave(false)
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
		OptionalUpdateCallback: defaultUpdateCallback(au.Enforcer),
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
	return au.Enforcer.SetWatcher(au.Watcher)
}

// SetAdapter sets the default adapter for the enforcer.
func SetAdapter(adapter persist.Adapter) {
	defaultAdapter = adapter
}

func (au *Authorization) CheckPermission(ctx context.Context, identity security.Identity, item *security.PermissionItem) (bool, error) {
	return au.Enforcer.Enforce(au.RequestParser(ctx, identity, item)...)
}

// SetDefaultAuthorization sets the default authorization.
func SetDefaultAuthorization(au *Authorization) {
	DefaultAuthorization = au
}

func defaultUpdateCallback(e casbin.IEnforcer) func(string) {
	return func(msg string) {
		msgStruct := &rediswatcher.MSG{}

		err := msgStruct.UnmarshalBinary([]byte(msg))
		if err != nil {
			log.Error(err)
			return
		}

		var res bool
		switch msgStruct.Method {
		case rediswatcher.Update, rediswatcher.UpdateForSavePolicy:
			err = e.LoadPolicy()
			res = true
		case rediswatcher.UpdateForAddPolicy:
			res, err = e.SelfAddPolicy(msgStruct.Sec, msgStruct.Ptype, msgStruct.NewRule)
		case rediswatcher.UpdateForAddPolicies:
			res, err = e.SelfAddPolicies(msgStruct.Sec, msgStruct.Ptype, msgStruct.NewRules)
		case rediswatcher.UpdateForRemovePolicy:
			res, err = e.SelfRemovePolicy(msgStruct.Sec, msgStruct.Ptype, msgStruct.NewRule)
		case rediswatcher.UpdateForRemoveFilteredPolicy:
			res, err = e.SelfRemoveFilteredPolicy(msgStruct.Sec, msgStruct.Ptype, msgStruct.FieldIndex, msgStruct.FieldValues...)
		case rediswatcher.UpdateForRemovePolicies:
			res, err = e.SelfRemovePolicies(msgStruct.Sec, msgStruct.Ptype, msgStruct.NewRules)
		case rediswatcher.UpdateForUpdatePolicy:
			res, err = e.SelfUpdatePolicy(msgStruct.Sec, msgStruct.Ptype, msgStruct.OldRule, msgStruct.NewRule)
		case rediswatcher.UpdateForUpdatePolicies:
			res, err = e.SelfUpdatePolicies(msgStruct.Sec, msgStruct.Ptype, msgStruct.OldRules, msgStruct.NewRules)
		default:
			err = errors.New("[woocoo-authz]update callback unknown update type")
		}
		if err != nil {
			log.Error(err)
		}
		if !res {
			log.Error("[woocoo-authz]callback update policy failed")
		}
	}
}

func defaultRequestParserFunc(ctx context.Context, identity security.Identity, item *security.PermissionItem) []any {
	return []any{identity.Name(), item.Action, item.Operator}
}
