package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/lmenezes/cerebro/internal/auth"
)

func (d *Deps) handleLogin(w http.ResponseWriter, r *http.Request) {
	var user, password string
	r.Body = http.MaxBytesReader(w, r.Body, d.Cfg.Server.MaxRequestBytes)
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/x-www-form-urlencoded") || strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid login form data", http.StatusBadRequest)
			return
		}
		user = r.FormValue("user")
		password = r.FormValue("password")
	} else {
		var payload struct {
			User     string `json:"user"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid login json data", http.StatusBadRequest)
			return
		}
		user = payload.User
		password = payload.Password
	}
	if user == "" || password == "" {
		d.auditAuth(r, "password_login_failed", auth.Identity{Username: user, Provider: "password"}, "failure", "missing credentials")
		http.Error(w, "invalid login form data", http.StatusBadRequest)
		return
	}
	identity, err := d.Auth.Authenticate(user, password)
	if err != nil {
		d.auditAuth(r, "password_login_failed", auth.Identity{Username: user, Provider: "password"}, "failure", "invalid credentials")
		http.Redirect(w, r, basePathFor(d, "/#/login?error=invalid"), http.StatusSeeOther)
		return
	}
	if err := d.Auth.SetSessionIdentity(w, r, identity); err != nil {
		d.auditAuth(r, "password_login_failed", identity, "failure", "session error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d.auditAuth(r, "password_login_succeeded", identity, "success", "")
	redirect := d.Auth.ConsumeRedirect(w, r)
	if redirect == "" {
		redirect = basePathFor(d, "/")
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther) // #nosec G710 -- redirect comes from auth.ConsumeRedirect, which accepts only same-origin absolute paths.
}
