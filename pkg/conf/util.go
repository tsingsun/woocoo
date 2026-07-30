package conf

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/go-envparse"
)

var (
	// envRegexp matches ${VAR}, ${ VAR }, ${VAR:-default}, and \${VAR} (escaped literal).
	envRegexp       = regexp.MustCompile(`\\?\$\{[ \t]*[A-Za-z_][A-Za-z0-9_.]*(?:[ \t]*:-[ \t]*[^}]*)?[ \t]*\}`)
	defaultEnvFiles = []string{".env", ".env.local"}
)

// TLS is the TLS configuration for TLS connections
// TLS content can be file or cert string,depend on your application access
// notice: TLS is experimental,and only support file path in configuration
type TLS struct {
	CA                 string `json:"ca" yaml:"ca"`
	Cert               string `json:"cert" yaml:"cert"`
	Key                string `json:"key" yaml:"key"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify" yaml:"insecureSkipVerify"`
}

// NewTLS creates a new TLS configuration. It will initialize the same defaults as the tls.Config struct.
func NewTLS(cnf *Configuration) *TLS {
	t := &TLS{}
	t.Apply(cnf)
	return t
}

func (t *TLS) Apply(cnf *Configuration) {
	if err := cnf.Unmarshal(t); err != nil {
		panic(err)
	}
	t.CA = cnf.Abs(t.CA)
	t.Cert = cnf.Abs(t.Cert)
	t.Key = cnf.Abs(t.Key)
}

// nolint:stylecheck
func (t *TLS) BuildTlsConfig() (*tls.Config, error) {
	tc := &tls.Config{
		InsecureSkipVerify: t.InsecureSkipVerify,
	}
	if t.CA != "" {
		caCert, err := os.ReadFile(t.CA)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to append CA certificate: %s", t.CA)
		}
		tc.RootCAs = caCertPool
	}
	if t.Cert != "" && t.Key != "" {
		cert, err := tls.LoadX509KeyPair(t.Cert, t.Key)
		if err != nil {
			return nil, err
		}
		tc.Certificates = []tls.Certificate{cert}
	}
	return tc, nil
}

// GetIP returns the first non-loopback address
func GetIP(useIPv6 bool) string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "error"
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if useIPv6 && ipnet.IP.To16() != nil {
				return ipnet.IP.To16().String()
			} else if ipnet.IP.To4() != nil {
				return ipnet.IP.To4().String()
			}
		}
	}
	panic("Unable to determine local IP address (non loopback).")
}

// TryLoadEnvFromFile try load env from files
func TryLoadEnvFromFile(scan, mod string) {
	files := make([]string, len(defaultEnvFiles))
	copy(files, defaultEnvFiles)
	if mod != "" {
		files = append(files, fmt.Sprintf(".env.%s", mod))
	}
	for _, name := range files {
		fp := filepath.Join(scan, name)
		if _, err := os.Stat(fp); err != nil {
			continue
		}
		// load env name
		bs, err := os.ReadFile(fp)
		if err != nil {
			continue
		}
		emp, err := envparse.Parse(bytes.NewBuffer(bs))
		if err != nil {
			log.Printf("load env file %s error: %s", fp, err)
			continue
		}
		// set env
		for k, v := range emp {
			err := os.Setenv(k, v)
			if err != nil {
				log.Printf("set env %s:%s error: %s", k, v, err)
			}
		}
	}
}

// ParseEnv parse env value in src.
//
// Supported syntax:
//   - ${VAR}          — replaced with env value, empty if not set
//   - ${VAR:-default} — replaced with env value, or "default" if not set/empty
//   - \${VAR}         — escaped, literal ${VAR} in output
//
// Variable names must start with [A-Za-z_], followed by [A-Za-z0-9_.].
func ParseEnv(src []byte) []byte {
	if !bytes.Contains(src, []byte("${")) {
		return src
	}
	return envRegexp.ReplaceAllFunc(src, func(match []byte) []byte {
		// \${...} → literal ${...}
		if match[0] == '\\' {
			return match[1:]
		}
		// strip ${ and }
		inner := string(bytes.TrimSpace(match[2 : len(match)-1]))

		name := inner
		defVal := ""
		hasDefault := false
		if idx := strings.Index(inner, ":-"); idx >= 0 {
			name = strings.TrimSpace(inner[:idx])
			defVal = strings.TrimSpace(inner[idx+2:])
			hasDefault = true
		}

		if val := os.Getenv(name); val != "" {
			return []byte(val)
		}
		if hasDefault {
			return []byte(defVal)
		}
		return nil
	})
}
