package auth

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/lmenezes/cerebro/internal/config"
)

type ProxyAuthenticator struct {
	providerID     string
	userHeader     string
	groupsHeader   string
	groupSeparator string
	defaultGroups  []string
	logoutURL      string
	trustedNets    []*net.IPNet
}

func NewProxyAuthenticator(settings config.ProxyAuth) (*ProxyAuthenticator, error) {
	return NewNamedProxyAuthenticator("", settings)
}

func NewNamedProxyAuthenticator(providerID string, settings config.ProxyAuth) (*ProxyAuthenticator, error) {
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
		providerID:     providerID,
		userHeader:     settings.UserHeader,
		groupsHeader:   settings.GroupsHeader,
		groupSeparator: separator,
		defaultGroups:  mergeGroups(settings.DefaultGroups),
		logoutURL:      strings.TrimSpace(settings.LogoutURL),
		trustedNets:    trustedNets,
	}, nil
}

func (p *ProxyAuthenticator) ProviderID() string {
	if p == nil {
		return ""
	}
	return p.providerID
}

func (p *ProxyAuthenticator) LogoutURL() string {
	if p == nil {
		return ""
	}
	return p.logoutURL
}

func (p *ProxyAuthenticator) Identity(r *http.Request) (Identity, bool) {
	if p == nil || !p.trustedRemote(r.RemoteAddr) {
		return Identity{}, false
	}
	username := strings.TrimSpace(r.Header.Get(p.userHeader))
	if username == "" {
		username = strings.TrimSpace(r.Header.Get("X-Forwarded-Preferred-Username"))
	}
	if username == "" {
		username = strings.TrimSpace(r.Header.Get("X-Forwarded-Email"))
	}
	if username == "" {
		return Identity{}, false
	}
	return Identity{
		Username:   username,
		Groups:     p.groups(r),
		Provider:   "proxy",
		ProviderID: p.providerID,
	}, true
}

func (p *ProxyAuthenticator) groups(r *http.Request) []string {
	groups := append([]string(nil), p.defaultGroups...)
	if p.groupsHeader != "" {
		for _, value := range r.Header.Values(p.groupsHeader) {
			for _, group := range strings.Split(value, p.groupSeparator) {
				groups = append(groups, group)
			}
		}
	}
	return mergeGroups(groups)
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
