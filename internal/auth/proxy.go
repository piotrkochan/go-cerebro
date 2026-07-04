package auth

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/lmenezes/cerebro/internal/config"
)

type ProxyAuthenticator struct {
	userHeader     string
	groupsHeader   string
	groupSeparator string
	trustedNets    []*net.IPNet
}

func NewProxyAuthenticator(settings config.ProxyAuth) (*ProxyAuthenticator, error) {
	if strings.TrimSpace(settings.UserHeader) == "" {
		return nil, fmt.Errorf("proxy auth requires user_header")
	}
	if len(settings.TrustedProxies) == 0 {
		return nil, fmt.Errorf("proxy auth requires trusted_proxies")
	}
	trustedNets := make([]*net.IPNet, 0, len(settings.TrustedProxies))
	for _, raw := range settings.TrustedProxies {
		network, err := parseTrustedProxy(raw)
		if err != nil {
			return nil, err
		}
		trustedNets = append(trustedNets, network)
	}
	separator := settings.GroupSeparator
	if separator == "" {
		separator = ","
	}
	return &ProxyAuthenticator{
		userHeader:     settings.UserHeader,
		groupsHeader:   settings.GroupsHeader,
		groupSeparator: separator,
		trustedNets:    trustedNets,
	}, nil
}

func (p *ProxyAuthenticator) Identity(r *http.Request) (Identity, bool) {
	if p == nil || !p.trustedRemote(r.RemoteAddr) {
		return Identity{}, false
	}
	username := strings.TrimSpace(r.Header.Get(p.userHeader))
	if username == "" {
		return Identity{}, false
	}
	return Identity{
		Username: username,
		Groups:   p.groups(r),
	}, true
}

func (p *ProxyAuthenticator) groups(r *http.Request) []string {
	if p.groupsHeader == "" {
		return nil
	}
	seen := map[string]bool{}
	out := []string{}
	for _, value := range r.Header.Values(p.groupsHeader) {
		for _, group := range strings.Split(value, p.groupSeparator) {
			group = strings.TrimSpace(group)
			if group == "" || seen[group] {
				continue
			}
			seen[group] = true
			out = append(out, group)
		}
	}
	return out
}

func (p *ProxyAuthenticator) trustedRemote(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(strings.TrimSpace(host))
	if ip == nil {
		return false
	}
	for _, network := range p.trustedNets {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func parseTrustedProxy(raw string) (*net.IPNet, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, fmt.Errorf("proxy auth trusted_proxies contains an empty entry")
	}
	if strings.Contains(value, "/") {
		_, network, err := net.ParseCIDR(value)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy %q: %w", raw, err)
		}
		return network, nil
	}
	ip := net.ParseIP(value)
	if ip == nil {
		return nil, fmt.Errorf("invalid trusted proxy %q", raw)
	}
	bits := 32
	if ip.To4() == nil {
		bits = 128
	}
	return &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)}, nil
}
