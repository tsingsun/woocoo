package conf

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
)

// TLS is the TLS configuration for TLS connections
// TLS content can be file or cert string,depend on your application access
// notice: TLS is experimental
type TLS struct {
	CA                 string `json:"ca" yaml:"ca"`
	Cert               string `json:"cert" yaml:"cert"`
	Key                string `json:"key" yaml:"key"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify" yaml:"insecureSkipVerify"`
}

// NewTLS creates a new TLS configuration. It will initialize the same defaults as the tls.Config struct.
func NewTLS(cfg *Configuration) *TLS {
	t := &TLS{}
	t.Apply(cfg)
	return t
}

func (t *TLS) Apply(cfg *Configuration) {
	if err := cfg.Unmarshal(t); err != nil {
		panic(err)
	}
	t.CA = cfg.Abs(t.CA)
	t.Cert = cfg.Abs(t.Cert)
	t.Key = cfg.Abs(t.Key)
}

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
	panic("Unable to determine local IP address (non loopback). Exiting.")
}
