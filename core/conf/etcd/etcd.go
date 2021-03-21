package etcd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/spf13/viper"
	"io"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func init() {
	viper.RemoteConfig = &etcdConfigProvider{}
}

type etcdConfigProvider struct {
	client *clientv3.Client
}

func (e etcdConfigProvider) Get(rp viper.RemoteProvider) (io.Reader, error) {
	val, err := e.etcdGet(rp)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(val), nil
}

func (e etcdConfigProvider) Watch(rp viper.RemoteProvider) (io.Reader, error) {
	val, err := e.etcdGet(rp)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(val), nil
}

func (e etcdConfigProvider) WatchChannel(rp viper.RemoteProvider) (<-chan *viper.RemoteResponse, chan bool) {
	cl, err := getEtcdClient(&e, rp)
	if err != nil {
		return nil, nil
	}
	quit := make(chan bool)
	quitwc := make(chan bool)
	viperResponsCh := make(chan *viper.RemoteResponse)
	rch := cl.Watch(context.Background(), rp.Path(), clientv3.WithPrefix())
	go func(rch *clientv3.WatchChan, vr chan<- *viper.RemoteResponse, quitwc <-chan bool, quit chan<- bool) {
		for {
			select {
			case <-quitwc:
				quit <- true
				return
			default:
				for n := range *rch {
					for _, ev := range n.Events {
						switch ev.Type {
						case mvccpb.PUT:
							viperResponsCh <- &viper.RemoteResponse{
								Error: n.Err(),
								Value: ev.Kv.Value,
							}
							log.Printf("%s config has changed", rp.Path())
						case mvccpb.DELETE:
							log.Printf("%s config has deleted", rp.Path())
							quit <- true
						}
					}
				}
			}

		}

	}(&rch, viperResponsCh, quitwc, quit)
	return viperResponsCh, quit
}

func (e etcdConfigProvider) etcdGet(rp viper.RemoteProvider) ([]byte, error) {
	cl, err := getEtcdClient(&e, rp)
	if err != nil {
		return nil, err
	}
	getResp, err := cl.Get(context.Background(), rp.Path(), clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	if len(getResp.Kvs) == 0 {
		return nil, errors.New("key's value is empty")
	}
	return getResp.Kvs[0].Value, nil

}

func getEtcdClient(e *etcdConfigProvider, rp viper.RemoteProvider) (client *clientv3.Client, err error) {
	// etcd has create client
	if e.client != nil {
		return e.client, nil
	}
	cnf := clientv3.Config{
		Endpoints:   []string{rp.Endpoint()},
		DialTimeout: 30 * time.Second,
	}
	client, err = clientv3.New(cnf)
	return
}

func parseConnString(connString string) (*clientv3.Config, error) {
	var err error
	schema := ""
	uri := connString
	cnf := &clientv3.Config{}
	if connString == "" {
		return cnf, nil
	}
	if idx := strings.Index(uri, "://"); idx != -1 {
		schema = uri[:idx+3]
		uri = uri[idx+4:]
	}
	if idx := strings.Index(uri, "@"); idx != -1 {
		userInfo := uri[:idx]
		uri = uri[idx+1:]

		username := userInfo
		var password string

		if idx := strings.Index(userInfo, ":"); idx != -1 {
			username = userInfo[:idx]
			password = userInfo[idx+1:]
		}

		if len(username) > 1 {
			if strings.Contains(username, "/") {
				return nil, fmt.Errorf("unescaped slash in username")
			}
		}

		cnf.Username, err = url.QueryUnescape(username)
		if err != nil {
			return nil, fmt.Errorf("invalid username: %w", err)
		}
		if len(password) > 1 {
			if strings.Contains(password, ":") {
				return nil, fmt.Errorf("unescaped colon in password")
			}
			if strings.Contains(password, "/") {
				return nil, fmt.Errorf("unescaped slash in password")
			}
			cnf.Password, err = url.QueryUnescape(password)
			if err != nil {
				return nil, fmt.Errorf("invalid password: %w", err)
			}
		}
	}

	// fetch the hosts field
	hosts := uri
	if idx := strings.IndexAny(uri, "/?@"); idx != -1 {
		if uri[idx] == '@' {
			return nil, fmt.Errorf("unescaped @ sign in user info")
		}
		if uri[idx] == '?' {
			return nil, fmt.Errorf("must have a / before the query")
		}
		hosts = uri[:idx]
	}

	parsedHosts := strings.Split(hosts, ",")
	for _, host := range parsedHosts {
		if host != "" {
			cnf.Endpoints = append(cnf.Endpoints, schema+host)
		}
	}
	uri = uri[len(hosts):]

	var connectionArgsFromTXT []string
	connectionArgsFromQueryString, err := extractQueryArgsFromURI(uri)
	connectionArgPairs := append(connectionArgsFromTXT, connectionArgsFromQueryString...)

	for _, pair := range connectionArgPairs {
		if err = addEtcdClientOption(cnf, pair); err != nil {
			return nil, err
		}
	}
	return cnf, nil
}

func addEtcdClientOption(cnf *clientv3.Config, pair string) error {
	kv := strings.SplitN(pair, "=", 2)
	if len(kv) != 2 || kv[0] == "" {
		return errors.New("connection option must be key=value: " + pair)
	}

	key, err := url.QueryUnescape(kv[0])
	if err != nil {
		return fmt.Errorf("invalid option key \"%s\": %w", kv[0], err)
	}

	value, err := url.QueryUnescape(kv[1])
	if err != nil {
		return fmt.Errorf("invalid option value \"%s\": %w", kv[1], err)
	}

	lowerKey := strings.ToLower(key)

	switch lowerKey {
	case "auto-sync-interval":
		if val, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("invalid option value \"%s\": %w", pair, err)
		} else {
			cnf.AutoSyncInterval = time.Duration(val) * time.Second
		}
	case "username":
		cnf.Username = value
	case "password":
		cnf.Password = value
	case "dial-timeout":
		if val, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("invalid option value \"%s\": %w", pair, err)
		} else {
			cnf.DialTimeout = time.Duration(val) * time.Second
		}
	case "dial-keep-alive-time":
		if val, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("invalid option value \"%s\": %w", pair, err)
		} else {
			cnf.DialKeepAliveTime = time.Duration(val) * time.Second
		}
	case "dial-keep-alive-timeout":
		if val, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("invalid option value \"%s\": %w", pair, err)
		} else {
			cnf.DialKeepAliveTimeout = time.Duration(val) * time.Second
		}
	case "reject-old-cluster":
		cnf.RejectOldCluster = value == "true"
	}
	return nil
}

func extractQueryArgsFromURI(uri string) ([]string, error) {
	if len(uri) == 0 {
		return nil, nil
	}

	if uri[0] != '?' {
		return nil, errors.New("must have a ? separator between path and query")
	}

	uri = uri[1:]
	if len(uri) == 0 {
		return nil, nil
	}
	return strings.FieldsFunc(uri, func(r rune) bool { return r == ';' || r == '&' }), nil

}

// init client for viper remote,the connection string will be like mongo.
// http://[username:password]@[host1:port],[host2:port]/?key1=value1&key2=value2
// example:
//
func UseSimple(connString string) error {
	ec, ok := viper.RemoteConfig.(*etcdConfigProvider)
	if !ok {
		panic("viper global remote config is not ectd provider")
	}
	cnf, err := parseConnString(connString)
	if err != nil {
		return err
	}
	ec.client, err = clientv3.New(*cnf)
	return err
}
