package conf

import (
	"fmt"
	"os"
	"path/filepath"
)

// configuration detail
// attachFiles: the files will merge into main configuration and override it.
type options struct {
	localPath      string
	attachFiles    []string
	remoteProvider remoteProvider
	isDebug        bool
	//use viper global
	global bool
}

var defaultOptions = options{
	localPath: filepath.Dir(os.Args[0]) + "./app.yaml",
	isDebug:   false,
	global:    true,
}

// the function to apply configuration option
type Option func(*options)

// init local instance file path
// A s is file path
func LocalPath(s string) Option {
	return func(o *options) {
		_, err := os.Stat(s)
		if err != nil {
			panic(fmt.Sprintf("local file '%s' is not exists", s))
		}
		o.localPath = s
	}
}

// AddSecureRemoteProvider adds a etcd configuration source.
// Secure Remote Providers are searched in the order they are added.
// provider is a string value: "etcd", "consul" or "firestore" are currently supported.
// endpoint is the url.  etcd requires http://ip:port  consul requires ip:port
// secretkeyring is the filepath to your openpgp secret keyring.  e.g. /etc/secrets/myring.gpg
// path is the path in the k/v store to retrieve configuration
// To retrieve a instance file called myapp.json from /configs/myapp.json
// you should set path to /configs and set instance name (SetConfigName()) to
// "myapp"
// Secure Remote Providers are implemented with github.com/bketelsen/crypt
// add import "_ github.com/spf13/viper/etcd" to use origin viper etcd which support etcd v2,consul.
// if you want to use etcd v3,add "gitee.com/woocoo/core/instance/etcd"
func RemoteProvider(provider, endpoint, path, secretkeyring string) Option {
	return func(o *options) {
		if provider == "" || endpoint == "" || path == "" {
			panic("a etcd provider must set a value to each parameter,such as below: provider,endpoint,path")
		}
		o.remoteProvider = remoteProvider{provider: provider, endpoint: endpoint, path: path, secretKeyring: secretkeyring}
	}
}

// 附加文件中的配置将会重写主配置文件,对于非法的文件,将被忽略.
// you can set a configuration for dev ENV,but attach instance only effect in local file configuration
func AttachFiles(paths ...string) Option {
	return func(o *options) {
		for _, s := range paths {
			_, err := os.Stat(s)
			if err != nil {
				panic(fmt.Errorf("attach config file %s error,%s", s, err))
			}
			o.attachFiles = append(o.attachFiles, s)
		}
	}
}
