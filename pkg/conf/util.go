package conf

import (
	"net"
)

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
