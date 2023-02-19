package authz

import (
	"errors"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
	rediswatcher "github.com/casbin/redis-watcher/v2"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"strings"
	"sync"
)

var (
	defaultAdapter persist.Adapter
	once           sync.Once

	defaultAuthorization *Authorization
)

type (
	// Authorization is an Authorization feature base on casbin.
	Authorization struct {
		Enforcer *casbin.Enforcer
	}
)

// NewAuthorization returns a new authenticator by application configuration.
// Configuration example:
//
//	authz:
//	  watcherOptions:
//	    options:
//	      addr: "localhost:6379"
//	      channel: "/casbin"
//	  model: /path/to/model.conf
//	  policy: /path/to/policy.csv
func NewAuthorization(cnf *conf.Configuration) *Authorization {
	opts := rediswatcher.WatcherOptions{}
	if cnf.IsSet("watcherOptions") {
		err := cnf.Sub("watcherOptions").Unmarshal(&opts)
		if err != nil {
			panic(err)
		}
	}
	var (
		w   persist.Watcher
		err error
	)
	if opts.Options.Addr != "" {
		w, err = rediswatcher.NewWatcher(opts.Options.Addr, opts)
	} else if opts.ClusterOptions.Addrs != nil {
		w, err = rediswatcher.NewWatcherWithCluster(opts.Options.Addr, opts)
	}
	if err != nil {
		panic(err)
	}
	// model
	var dsl, policy any
	m := cnf.String("model")
	if strings.ContainsRune(m, '\n') {
		dsl, err = model.NewModelFromString(m)
		if err != nil {
			panic(err)
		}
	} else {
		dsl = cnf.String("model")
	}
	// policy
	if pv := cnf.String("policy"); pv != "" {
		SetAdapter(fileadapter.NewAdapter(pv))
	}
	policy = defaultAdapter
	enforer, err := casbin.NewEnforcer(dsl, policy)
	if err != nil {
		panic(err)
	}

	if w != nil { // enable watcher
		if err := enforer.SetWatcher(w); err != nil {
			panic(err)
		}
		if err := w.SetUpdateCallback(defaultUpdateCallback(enforer)); err != nil {
			panic(err)
		}
	}
	return &Authorization{
		Enforcer: enforer,
	}
}

// SetAdapter sets the default adapter for the enforcer.
func SetAdapter(adapter persist.Adapter) {
	defaultAdapter = adapter
}

func SetDefaultAuthorization(cnf *conf.Configuration) *Authorization {
	once.Do(func() {
		defaultAuthorization = NewAuthorization(cnf)
	})
	return defaultAuthorization
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
			err = errors.New("unknown update type")
		}
		if err != nil {
			log.Error(err)
		}
		if !res {
			log.Error("callback update policy failed")
		}
	}
}
