package server

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"github.com/lmenezes/cerebro/internal/auth"
	"github.com/lmenezes/cerebro/internal/version"
)

//go:embed templates/login.html
var templatesFS embed.FS

//go:embed all:static
var staticFS embed.FS

var embeddedStatic = mustSub(staticFS, "static")

var loginTmpl = template.Must(template.ParseFS(templatesFS, "templates/login.html"))

// indexHandler enforces auth (when enabled, redirects to /login) and serves the React shell.
func indexHandler(authMod *auth.Module) http.HandlerFunc {
	assets := publicAssets()
	return func(w http.ResponseWriter, r *http.Request) {
		if authMod.Enabled() {
			if _, ok := authMod.SessionUser(r); !ok {
				_ = authMod.SetRedirectIfSafe(w, r, r.URL.RequestURI())
				http.Redirect(w, r, "login", http.StatusSeeOther)
				return
			}
		}
		request := r.Clone(r.Context())
		request.URL.Path = "/index.html"
		assets.ServeHTTP(w, request)
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
			http.Redirect(w, r, redirect, http.StatusSeeOther) // #nosec G710 -- redirect comes from auth.ConsumeRedirect, which accepts only same-origin absolute paths.
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = loginTmpl.Execute(w, map[string]string{"Version": version.Version, "Message": ""})
	}
}

// publicAssets serves embedded frontend assets.
func publicAssets() http.Handler {
	root := http.FileServer(http.FS(embeddedStatic))
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

func mustSub(root fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(root, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

func embeddedFS() fs.FS { return embeddedStatic }
