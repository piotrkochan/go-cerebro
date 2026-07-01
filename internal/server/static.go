package server

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/lmenezes/cerebro/internal/auth"
)

//go:embed all:static
var staticFS embed.FS

var embeddedStatic = mustSub(staticFS, "static")

// indexHandler serves the React shell. The React app asks /auth/status and routes
// unauthenticated users to its login page.
func indexHandler(authMod *auth.Module) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveIndex(w, r)
	}
}

// loginHandler sends legacy /login URLs to the React login route.
func loginHandler(authMod *auth.Module) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authMod.Enabled() {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if _, ok := authMod.SessionUser(r); ok {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/#/login", http.StatusSeeOther)
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

func serveIndex(w http.ResponseWriter, r *http.Request) {
	content, err := fs.ReadFile(embeddedStatic, "index.html")
	if err != nil {
		http.Error(w, "frontend index not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(content))
}

func mustSub(root fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(root, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

func embeddedFS() fs.FS { return embeddedStatic }
