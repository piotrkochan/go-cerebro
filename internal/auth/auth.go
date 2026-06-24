package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/lmenezes/cerebro/internal/config"
)

const (
	SessionName    = "cerebro"
	SessionUserKey = "username"
	RedirectURL    = "redirect"
)

type Service interface {
	Authenticate(username, password string) (string, error)
}

var ErrInvalidCredentials = errors.New("invalid credentials")

type ctxUserKey struct{}

func WithUser(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, ctxUserKey{}, user)
}

func UserFrom(ctx context.Context) string {
	if v := ctx.Value(ctxUserKey{}); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

type Module struct {
	enabled bool
	service Service
	store   *sessions.CookieStore
}

func NewModule(cfg *config.Config) (*Module, error) {
	store := sessions.NewCookieStore([]byte(cfg.Server.Secret))
	store.Options = &sessions.Options{
		Path:     strings.TrimRight(cfg.Server.BasePath, "/") + "/",
		HttpOnly: true,
		MaxAge:   0,
		SameSite: http.SameSiteLaxMode,
		Secure:   cfg.Server.CookieSecure,
	}
	if store.Options.Path == "" {
		store.Options.Path = "/"
	}

	m := &Module{store: store}
	switch cfg.Auth.Type {
	case "", "disabled", "none":
		m.enabled = false
	case "basic":
		svc, err := NewBasicService(cfg.Auth.Settings)
		if err != nil {
			return nil, err
		}
		m.enabled = true
		m.service = svc
	case "ldap":
		svc, err := NewLDAPService(cfg.Auth.Settings)
		if err != nil {
			return nil, err
		}
		m.enabled = true
		m.service = svc
	default:
		return nil, fmt.Errorf("unknown auth type: %s", cfg.Auth.Type)
	}
	return m, nil
}

func (m *Module) Enabled() bool { return m.enabled }

func (m *Module) Store() *sessions.CookieStore { return m.store }

func (m *Module) Authenticate(username, password string) (string, error) {
	if m.service == nil {
		return "", errors.New("authentication not enabled")
	}
	return m.service.Authenticate(username, password)
}

func (m *Module) SessionUser(r *http.Request) (string, bool) {
	sess, err := m.store.Get(r, SessionName)
	if err != nil {
		return "", false
	}
	v, ok := sess.Values[SessionUserKey]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func (m *Module) SetSessionUser(w http.ResponseWriter, r *http.Request, username string) error {
	sess, _ := m.store.Get(r, SessionName)
	sess.Values[SessionUserKey] = username
	delete(sess.Values, RedirectURL)
	return sess.Save(r, w)
}

func (m *Module) ClearSession(w http.ResponseWriter, r *http.Request) error {
	sess, _ := m.store.Get(r, SessionName)
	sess.Options.MaxAge = -1
	for k := range sess.Values {
		delete(sess.Values, k)
	}
	return sess.Save(r, w)
}

func (m *Module) SetRedirect(w http.ResponseWriter, r *http.Request, uri string) error {
	sess, _ := m.store.Get(r, SessionName)
	sess.Values[RedirectURL] = uri
	return sess.Save(r, w)
}

// ConsumeRedirect returns the previously stored redirect URL and clears it. The returned
// path is validated to be a same-origin absolute path so attackers can't smuggle a
// protocol-relative URL (e.g. "//evil.com/x") through a malformed initial request.
func (m *Module) ConsumeRedirect(w http.ResponseWriter, r *http.Request) string {
	sess, _ := m.store.Get(r, SessionName)
	v, ok := sess.Values[RedirectURL]
	if !ok {
		return ""
	}
	delete(sess.Values, RedirectURL)
	_ = sess.Save(r, w)
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return safeRedirect(s)
}

// SetRedirect stores the next-page URL after login, but only if it is a safe same-origin path.
func (m *Module) SetRedirectIfSafe(w http.ResponseWriter, r *http.Request, uri string) error {
	if safeRedirect(uri) == "" {
		return nil
	}
	return m.SetRedirect(w, r, uri)
}

// safeRedirect returns u only if it is a same-origin absolute path ("/something"). Anything
// else (empty, protocol-relative "//", absolute URL "http://", or backslash variant "/\")
// is rejected to prevent open redirect.
func safeRedirect(u string) string {
	if len(u) == 0 || u[0] != '/' {
		return ""
	}
	if len(u) >= 2 && (u[1] == '/' || u[1] == '\\') {
		return ""
	}
	return u
}

// Middleware enforces authentication for API endpoints. When auth is disabled it's a no-op.
// When auth is enabled and there is no session, it returns 303 for API endpoints so
// the UI can route the user to the login page without using a response envelope.
func (m *Module) APIMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled {
			next.ServeHTTP(w, r)
			return
		}
		user, ok := m.SessionUser(r)
		if !ok {
			w.WriteHeader(http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), user)))
	})
}
