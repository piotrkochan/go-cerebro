package server

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/lmenezes/cerebro/internal/auth"
	"github.com/lmenezes/cerebro/internal/version"
)

//go:embed templates/login.html
var templatesFS embed.FS

var loginTmpl = template.Must(template.ParseFS(templatesFS, "templates/login.html"))

// indexHandler enforces auth (when enabled, redirects to /login) and serves the React shell.
func indexHandler(authMod *auth.Module, publicDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if authMod.Enabled() {
			if _, ok := authMod.SessionUser(r); !ok {
				_ = authMod.SetRedirectIfSafe(w, r, r.URL.RequestURI())
				http.Redirect(w, r, "login", http.StatusSeeOther)
				return
			}
		}
		http.ServeFile(w, r, filepath.Join(publicDir, "index.html"))
	}
}

// loginHandler renders the login form.
func loginHandler(authMod *auth.Module) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authMod.Enabled() {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		// If already logged in, follow stored redirect or go to /.
		if _, ok := authMod.SessionUser(r); ok {
			redirect := authMod.ConsumeRedirect(w, r)
			if redirect == "" {
				redirect = "/"
			}
			http.Redirect(w, r, redirect, http.StatusSeeOther)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = loginTmpl.Execute(w, map[string]string{"Version": version.Version, "Message": ""})
	}
}

// publicAssets serves files from the public/ directory at the project root with sensible MIME guessing.
// We mount the OS filesystem (not embedded) so dev hot-reload of JS/CSS doesn't require rebuilds.
func publicAssets(publicDir string) http.Handler {
	root := http.FileServer(http.Dir(publicDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip any query string for mime detection.
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" || path == "index.html" {
			http.NotFound(w, r)
			return
		}
		root.ServeHTTP(w, r)
	})
}

// embeddedFS exposes the embedded templates/static files; reserved for future use if we choose to embed public/.
func embeddedFS() fs.FS { return templatesFS }
