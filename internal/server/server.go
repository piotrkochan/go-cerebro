package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/lmenezes/cerebro/internal/api"
	"github.com/lmenezes/cerebro/internal/auth"
	"github.com/lmenezes/cerebro/internal/config"
	"github.com/lmenezes/cerebro/internal/elastic"
	"github.com/lmenezes/cerebro/internal/history"
	"github.com/lmenezes/cerebro/internal/rbac"
)

type Server struct {
	cfg     *config.Config
	router  chi.Router
	humaAPI huma.API
	addr    string
}

type Options struct {
	Cfg     *config.Config
	Client  elastic.Client
	History *history.Store
	Auth    *auth.Module
}

func New(opts Options) *Server {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	if opts.Cfg.Logging.RequestLogEnabled {
		r.Use(requestLogger)
	}
	r.Use(middleware.Recoverer)
	r.Use(securityHeaders(opts.Cfg.Server))
	r.Use(maxRequestBody(opts.Cfg.Server.MaxRequestBytes))
	r.Use(middleware.Compress(5))
	r.Use(injectHTTPRequest)
	authorizer := rbac.New(opts.Cfg.RBAC)
	// Auth gate for API endpoints: requires a session cookie when auth is enabled.
	r.Use(apiAuthGate(opts.Auth, opts.Cfg.Server, authorizer, opts.Cfg.Logging.AuthLogEnabled))

	cfg := huma.DefaultConfig("Cerebro", "0.0.0")
	cfg.OpenAPI.Info.Description = "Cerebro — Elasticsearch cluster management UI."
	humaAPI := humachi.New(r, cfg)

	deps := &api.Deps{
		Cfg:     opts.Cfg,
		Client:  opts.Client,
		History: opts.History,
		Auth:    opts.Auth,
		RBAC:    authorizer,
	}

	clusterAPI := huma.NewGroup(humaAPI, "/clusters/{cluster}")
	clusterAPI.UseMiddleware(deps.ClusterTargetMiddleware)
	deps.RegisterAliases(clusterAPI)
	deps.RegisterCat(clusterAPI)
	deps.RegisterNodes(clusterAPI)
	deps.RegisterNavbar(clusterAPI)
	deps.RegisterCommons(clusterAPI)
	deps.RegisterTemplates(clusterAPI)
	deps.RegisterRepositories(clusterAPI)
	deps.RegisterSnapshots(clusterAPI)
	deps.RegisterAnalysis(clusterAPI)
	deps.RegisterClusterSettings(clusterAPI)
	deps.RegisterIndexSettings(clusterAPI)
	deps.RegisterCreateIndex(clusterAPI)
	deps.RegisterDataExplorer(clusterAPI)
	deps.RegisterDataStreams(clusterAPI)
	deps.RegisterILM(clusterAPI)
	deps.RegisterClusterChanges(clusterAPI)
	deps.RegisterConnect(humaAPI)
	deps.RegisterOverview(clusterAPI)
	deps.RegisterRest(clusterAPI)
	deps.RegisterAuth(humaAPI, &chiMux{r: r})

	// Static + login screen — served outside Huma.
	r.Get("/login", loginHandler(opts.Cfg, opts.Auth))
	r.Get("/", indexHandler(opts.Auth))

	// Wildcard GET fallback — chi only reaches here when no specific route matched.
	// We let http.FileServer answer 404 for missing files; no need to second-guess paths.
	publicHandler := publicAssets()
	r.Get("/*", publicHandler.ServeHTTP)

	router := mountBasePath(opts.Cfg.Server.BasePath, r)

	return &Server{
		cfg:     opts.Cfg,
		router:  router,
		humaAPI: humaAPI,
		addr:    fmt.Sprintf(":%d", opts.Cfg.Server.Port),
	}
}

func (s *Server) Handler() http.Handler { return s.router }

func (s *Server) HumaAPI() huma.API { return s.humaAPI }

func (s *Server) Run(ctx context.Context) error {
	srv := &http.Server{
		Addr:              s.addr,
		Handler:           s.router,
		TLSConfig:         serverTLSConfig(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MiB
	}
	errCh := make(chan error, 1)
	go func() {
		if s.cfg.Server.TLSCertFile != "" {
			errCh <- srv.ListenAndServeTLS(s.cfg.Server.TLSCertFile, s.cfg.Server.TLSKeyFile)
			return
		}
		errCh <- srv.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		return srv.Shutdown(context.Background())
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

func (s *Server) Scheme() string {
	if s.cfg.Server.TLSCertFile != "" {
		return "https"
	}
	return "http"
}

func serverTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
}

// chiMux adapts chi.Router to the small interface expected by api.RegisterAuth.
type chiMux struct {
	r chi.Router
}

func (m *chiMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	parts := strings.SplitN(pattern, " ", 2)
	if len(parts) == 2 {
		m.r.Method(parts[0], parts[1], http.HandlerFunc(handler))
		return
	}
	m.r.HandleFunc(pattern, handler)
}

// injectHTTPRequest stores the *http.Request on the context so handlers can read whitelisted headers.
func injectHTTPRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(api.WithHTTPRequest(r.Context(), r)))
	})
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}
		slog.InfoContext(r.Context(), "http request",
			"request_id", middleware.GetReqID(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"bytes", ww.BytesWritten(),
			"duration", time.Since(start).String(),
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)
	})
}

func securityHeaders(serverCfg config.Server) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'; object-src 'none'")
			h.Set("Referrer-Policy", "no-referrer")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			if serverCfg.HSTSEnabled && (r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")) {
				value := fmt.Sprintf("max-age=%d", serverCfg.HSTSMaxAgeSeconds)
				if serverCfg.HSTSIncludeSubDomains {
					value += "; includeSubDomains"
				}
				h.Set("Strict-Transport-Security", value)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func maxRequestBody(maxBytes int64) func(http.Handler) http.Handler {
	if maxBytes <= 0 {
		maxBytes = config.DefaultMaxRequestBytes
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// apiAuthGate runs the auth check only for combinations of method+path that the Cerebro API
// actually handles — every POST under an API prefix plus the single authenticated GET
// (/connect/hosts). All other GETs (HTML partials served from public/, /openapi.json, /, /login,
// static assets) pass through. /auth/login is explicitly excluded so users can authenticate.
func apiAuthGate(authMod *auth.Module, serverCfg config.Server, authorizer *rbac.Authorizer, authLogEnabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !shouldGate(r) {
				next.ServeHTTP(w, r)
				return
			}
			if authorizer.Enabled() && !authMod.Enabled() {
				auditAccess(r, "authentication_required", auth.Identity{}, "failure", "rbac requires authentication", authLogEnabled)
				writeProblem(w, r, serverCfg, http.StatusUnauthorized, "authentication required")
				return
			}
			if authMod.Enabled() {
				identity, ok := authMod.SessionIdentity(r)
				if !ok {
					auditAccess(r, "authentication_required", auth.Identity{}, "failure", "missing or expired session", authLogEnabled)
					writeProblem(w, r, serverCfg, http.StatusUnauthorized, "authentication required")
					return
				}
				r = r.WithContext(auth.WithIdentity(r.Context(), identity))
			}
			if serverCfg.CSRFEnabled {
				if !validRequestOrigin(serverCfg, r) {
					auditAccess(r, "csrf_rejected", auth.IdentityFrom(r.Context()), "failure", "invalid request origin", authLogEnabled)
					writeProblem(w, r, serverCfg, http.StatusForbidden, "invalid request origin")
					return
				}
				if requiresCSRFToken(r) && !authMod.ValidCSRF(r) {
					auditAccess(r, "csrf_rejected", auth.IdentityFrom(r.Context()), "failure", "invalid csrf token", authLogEnabled)
					writeProblem(w, r, serverCfg, http.StatusForbidden, "invalid csrf token")
					return
				}
			}
			if authorizer.Enabled() {
				subject := auth.UserFrom(r.Context())
				rbacRequest := rbac.Classify(r.Method, r.URL.Path)
				if !authorizer.Allow(subject, auth.GroupsFrom(r.Context()), rbacRequest) {
					auditAccess(r, "rbac_denied", auth.IdentityFrom(r.Context()), "failure", fmt.Sprintf("%s:%s:%s", rbacRequest.Resource, rbacRequest.Action, rbacRequest.Object), authLogEnabled)
					writeProblem(w, r, serverCfg, http.StatusForbidden, "Permission denied.")
					return
				}
			}
			_ = authMod.TouchSession(w, r)
			next.ServeHTTP(w, r)
		})
	}
}

type problemResponse struct {
	Schema string `json:"$schema,omitempty"`
	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

func writeProblem(w http.ResponseWriter, r *http.Request, serverCfg config.Server, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.Header().Set("Link", fmt.Sprintf("<%s>; rel=\"describedBy\"", schemaURL(serverCfg, r, "ErrorModel")))
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problemResponse{
		Schema: schemaURL(serverCfg, r, "ErrorModel"),
		Status: status,
		Title:  http.StatusText(status),
		Detail: detail,
	})
}

func schemaURL(serverCfg config.Server, r *http.Request, name string) string {
	basePath := strings.TrimRight(serverCfg.BasePath, "/")
	if basePath == "/" {
		basePath = ""
	}
	return requestOrigin(serverCfg, r) + basePath + "/schemas/" + name + ".json"
}

func auditAccess(r *http.Request, event string, identity auth.Identity, outcome, reason string, enabled bool) {
	if !enabled {
		return
	}
	attrs := []any{
		"event", event,
		"outcome", outcome,
		"user", identity.Username,
		"provider", identity.Provider,
		"provider_id", identity.ProviderID,
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
	}
	if reason != "" {
		attrs = append(attrs, "reason", reason)
	}
	slog.InfoContext(r.Context(), "auth audit", attrs...)
}

func validRequestOrigin(serverCfg config.Server, r *http.Request) bool {
	switch strings.ToLower(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site"))) {
	case "", "same-origin", "same-site", "none":
	default:
		return false
	}
	origin := r.Header.Get("Origin")
	if origin != "" && !sameOriginValue(serverCfg, origin, r) {
		return false
	}
	referer := r.Header.Get("Referer")
	if referer != "" && !sameOriginValue(serverCfg, referer, r) {
		return false
	}
	return true
}

func sameOriginValue(serverCfg config.Server, value string, r *http.Request) bool {
	expected := requestOrigin(serverCfg, r)
	return value == expected || strings.HasPrefix(value, expected+"/")
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

func shouldGate(r *http.Request) bool {
	if r.URL.Path == "/auth/login" {
		return false
	}
	if strings.HasPrefix(r.URL.Path, "/clusters/") {
		return true
	}
	switch r.Method {
	case http.MethodGet:
		// Only API GETs that expose authenticated application data are gated here. Other GETs
		// (React shell, static assets, /openapi.json, login/callback routes) must remain public
		// so the frontend and external login flows can boot.
		return r.URL.Path == "/connect/hosts" || r.URL.Path == "/auth/me"
	case http.MethodPost:
		return isAPI(r.URL.Path)
	}
	return false
}

func requiresCSRFToken(r *http.Request) bool {
	if r.Method == http.MethodGet {
		return r.URL.Path != "/auth/me" && r.URL.Path != "/connect/hosts"
	}
	return true
}

func mountBasePath(basePath string, app chi.Router) chi.Router {
	base := normalizedBasePath(basePath)
	if base == "/" {
		return app
	}
	outer := chi.NewRouter()
	outer.Mount(base, app)
	outer.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, base+"/", http.StatusSeeOther)
	})
	return outer
}

func normalizedBasePath(basePath string) string {
	base := "/" + strings.Trim(strings.TrimSpace(basePath), "/")
	if base == "/" {
		return "/"
	}
	return strings.TrimRight(base, "/")
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
		return nil, fmt.Errorf("invalid IP")
	}
	bits := 32
	if ip.To4() == nil {
		bits = 128
	}
	return &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)}, nil
}

// API paths that match exactly. Prefix-style matching would make future static files with API-like
// names accidentally require an API session.
var apiExact = map[string]bool{
	"/openapi.json":     true,
	"/openapi.yaml":     true,
	"/cat":              true,
	"/nodes":            true,
	"/navbar":           true,
	"/cluster_changes":  true,
	"/cluster_settings": true,
	"/index_settings":   true,
	"/connect":          true,
	"/repositories":     true,
	"/snapshots":        true,
	"/overview":         true,
	"/rest":             true,
	"/templates":        true,
}

// API path prefixes — each ends with "/" so HTML partials at the same root (e.g. /connect.html)
// are NOT matched.
var apiPrefixes = []string{
	"/docs/",
	"/aliases/",
	"/overview/",
	"/rest/",
	"/templates/",
	"/snapshots/",
	"/analysis/",
	"/cluster_settings/",
	"/cluster_changes/",
	"/index_settings/",
	"/create_index/",
	"/commons/",
	"/connect/",
	"/repositories/",
	"/auth/",
}

func isAPI(p string) bool {
	if apiExact[p] {
		return true
	}
	for _, pre := range apiPrefixes {
		if strings.HasPrefix(p, pre) {
			return true
		}
	}
	return false
}
