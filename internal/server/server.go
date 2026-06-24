package server

import (
	"context"
	"fmt"
	"net/http"
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
)

type Server struct {
	cfg     *config.Config
	router  chi.Router
	humaAPI huma.API
	addr    string
}

type Options struct {
	Cfg       *config.Config
	Client    elastic.Client
	History   *history.Store
	Auth      *auth.Module
	PublicDir string
}

func New(opts Options) *Server {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(securityHeaders)
	r.Use(maxRequestBody(opts.Cfg.Server.MaxRequestBytes))
	r.Use(middleware.Compress(5))
	r.Use(injectHTTPRequest)
	// Auth gate for API endpoints: requires a session cookie when auth is enabled.
	r.Use(apiAuthGate(opts.Auth))

	cfg := huma.DefaultConfig("Cerebro", "0.0.0")
	cfg.OpenAPI.Info.Description = "Cerebro — Elasticsearch cluster management UI."
	humaAPI := humachi.New(r, cfg)

	deps := &api.Deps{
		Cfg:     opts.Cfg,
		Client:  opts.Client,
		History: opts.History,
		Auth:    opts.Auth,
	}

	deps.RegisterAliases(humaAPI)
	deps.RegisterCat(humaAPI)
	deps.RegisterNodes(humaAPI)
	deps.RegisterNavbar(humaAPI)
	deps.RegisterCommons(humaAPI)
	deps.RegisterTemplates(humaAPI)
	deps.RegisterRepositories(humaAPI)
	deps.RegisterSnapshots(humaAPI)
	deps.RegisterAnalysis(humaAPI)
	deps.RegisterClusterSettings(humaAPI)
	deps.RegisterIndexSettings(humaAPI)
	deps.RegisterCreateIndex(humaAPI)
	deps.RegisterClusterChanges(humaAPI)
	deps.RegisterConnect(humaAPI)
	deps.RegisterOverview(humaAPI)
	deps.RegisterRest(humaAPI)
	deps.RegisterAuth(humaAPI, &chiMux{r: r})

	publicDir := opts.PublicDir
	if publicDir == "" {
		publicDir = "public"
	}

	// Static + login screen — served outside Huma.
	r.Get("/login", loginHandler(opts.Auth))
	r.Get("/", indexHandler(opts.Auth, publicDir))

	// Wildcard GET fallback — chi only reaches here when no specific route matched.
	// We let http.FileServer answer 404 for missing files; no need to second-guess paths.
	publicHandler := publicAssets(publicDir)
	r.Get("/*", publicHandler.ServeHTTP)

	return &Server{
		cfg:     opts.Cfg,
		router:  r,
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
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MiB
	}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
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

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'; object-src 'none'")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
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
// static assets) pass through. /auth/login and /auth/logout are explicitly excluded so users can
// authenticate.
func apiAuthGate(authMod *auth.Module) func(http.Handler) http.Handler {
	wrapped := authMod.APIMiddleware
	return func(next http.Handler) http.Handler {
		gated := wrapped(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !shouldGate(r) {
				next.ServeHTTP(w, r)
				return
			}
			gated.ServeHTTP(w, r)
		})
	}
}

func shouldGate(r *http.Request) bool {
	if r.URL.Path == "/auth/login" || r.URL.Path == "/auth/logout" {
		return false
	}
	switch r.Method {
	case http.MethodGet:
		// The only authenticated GET is /connect/hosts. Every other GET (HTML partials, statics,
		// /openapi.json) must remain publicly readable so the React frontend can boot.
		return r.URL.Path == "/connect/hosts"
	case http.MethodPost:
		return isAPI(r.URL.Path)
	}
	return false
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
