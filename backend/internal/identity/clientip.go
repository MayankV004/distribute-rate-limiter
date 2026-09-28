package identity

import (
	"net"
	"net/http"
	"strings"
)

type IPExtractor struct {
	trustedCIDRs []*net.IPNet
}

func NewIPExtractor(cidrList []string) (*IPExtractor, error) {
	var trusted []*net.IPNet
	for _, cidr := range cidrList {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, err
		}
		trusted = append(trusted, ipNet)
	}
	return &IPExtractor{trustedCIDRs: trusted}, nil
}

func (e *IPExtractor) isTrusted(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, cidr := range e.trustedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func (e *IPExtractor) Extract(r *http.Request) (string, bool) {
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteIP = r.RemoteAddr
	}

	if e.isTrusted(remoteIP) {
		xff := r.Header.Get("X-Forwarded-For")
		if xff != "" {
			parts := strings.Split(xff, ",")
			for i := len(parts) - 1; i >= 0; i-- {
				ip := strings.TrimSpace(parts[i])
				if !e.isTrusted(ip) {
					return ip, true
				}
			}
		}
	}

	return remoteIP, true
}
