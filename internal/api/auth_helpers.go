package api

import (
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/lmenezes/cerebro/internal/auth"
	"github.com/lmenezes/cerebro/internal/config"
)

func (d *Deps) auditAuth(r *http.Request, event string, identity auth.Identity, outcome, reason string) {
	if d.Cfg != nil && !d.Cfg.Logging.AuthLogEnabled {
		return
	}
	attrs := []any{
		"event", event,
		"outcome", outcome,
		"user", identity.Username,
		"provider", identity.Provider,
		"provider_id", identity.ProviderID,
		"remote_addr", r.RemoteAddr,
	}
	if reason != "" {
		attrs = append(attrs, "reason", reason)
	}
	slog.InfoContext(r.Context(), "auth audit", attrs...)
}

func basePathFor(d *Deps, suffix string) string {
	prefix := strings.TrimRight(d.Cfg.Server.BasePath, "/")
	if prefix == "" {
		return suffix
	}
	return prefix + suffix
}

func apiSchemaURL(serverCfg config.Server, r *http.Request, name string) string {
	basePath := strings.TrimRight(serverCfg.BasePath, "/")
	if basePath == "/" {
		basePath = ""
	}
	return requestOrigin(serverCfg, r) + basePath + "/schemas/" + name + ".json"
}

func requestOrigin(serverCfg config.Server, r *http.Request) string {
	if serverCfg.PublicURL != "" {
		if origin := publicOrigin(serverCfg.PublicURL); origin != "" {
			return origin
		}
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host
	if trustedRemote(r.RemoteAddr, serverCfg.TrustedProxies) {
		if forwardedProto := forwardedProto(r); forwardedProto != "" {
			scheme = strings.ToLower(forwardedProto)
		}
		if forwardedHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwardedHost != "" {
			host = forwardedHost
		}
	}
	return scheme + "://" + host
}

func forwardedProto(r *http.Request) string {
	value := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")))
	if value == "http" || value == "https" {
		return value
	}
	return ""
}

func publicOrigin(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func trustedRemote(remoteAddr string, trustedProxies []string) bool {
	if len(trustedProxies) == 0 {
		return false
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(strings.TrimSpace(host))
	if ip == nil {
		return false
	}
	for _, raw := range trustedProxies {
		network, err := trustedProxyNet(raw)
		if err == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}

func trustedProxyNet(raw string) (*net.IPNet, error) {
	value := strings.TrimSpace(raw)
	if strings.Contains(value, "/") {
		_, network, err := net.ParseCIDR(value)
		return network, err
	}
	ip := net.ParseIP(value)
	if ip == nil {
		return nil, http.ErrNotSupported
	}
	bits := 32
	if ip.To4() == nil {
		bits = 128
	}
	return &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)}, nil
}
