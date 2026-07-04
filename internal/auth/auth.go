package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/lmenezes/cerebro/internal/config"
)

const (
	SessionName      = "cerebro"
	SessionUserKey   = "username"
	SessionGroupsKey = "groups"
	SessionCSRFKey   = "csrf"
	RedirectURL      = "redirect"
)

type Service interface {
	Authenticate(username, password string) (Identity, error)
}

type Identity struct {
	Username string
	Groups   []string
}

var ErrInvalidCredentials = errors.New("invalid credentials")

type ctxIdentityKey struct{}

func WithUser(ctx context.Context, user string) context.Context {
	return WithIdentity(ctx, Identity{Username: user})
}

func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, ctxIdentityKey{}, identity)
}

func UserFrom(ctx context.Context) string {
	return IdentityFrom(ctx).Username
}

func GroupsFrom(ctx context.Context) []string {
	return IdentityFrom(ctx).Groups
}

func IdentityFrom(ctx context.Context) Identity {
	if v := ctx.Value(ctxIdentityKey{}); v != nil {
		if identity, ok := v.(Identity); ok {
			return identity
		}
	}
	return Identity{}
}

type Module struct {
	enabled  bool
	proxies  []*ProxyAuthenticator
	services []Service
	store    *sessions.CookieStore
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
	if cfg.Auth.Basic.Enabled {
		svc, err := NewBasicService(cfg.Auth.Basic)
		if err != nil {
			return nil, err
		}
		m.services = append(m.services, svc)
	}
	if cfg.Auth.LDAP.Enabled {
		svc, err := NewLDAPService(cfg.Auth.LDAP)
		if err != nil {
			return nil, err
		}
		m.services = append(m.services, svc)
	}
	if cfg.Auth.Proxy.Enabled {
		proxy, err := NewProxyAuthenticator(cfg.Auth.Proxy)
		if err != nil {
			return nil, err
		}
		m.proxies = append(m.proxies, proxy)
	}
	m.enabled = len(m.services) > 0 || len(m.proxies) > 0
	return m, nil
}

func (m *Module) Enabled() bool { return m.enabled }

func (m *Module) Store() *sessions.CookieStore { return m.store }

func (m *Module) Authenticate(username, password string) (Identity, error) {
	if len(m.services) == 0 {
		return Identity{}, errors.New("authentication not enabled")
	}
	for _, service := range m.services {
		identity, err := service.Authenticate(username, password)
		if err == nil {
			return identity, nil
		}
		if !errors.Is(err, ErrInvalidCredentials) {
			return Identity{}, err
		}
	}
	return Identity{}, ErrInvalidCredentials
}

func (m *Module) SessionUser(r *http.Request) (string, bool) {
	identity, ok := m.SessionIdentity(r)
	return identity.Username, ok
}

func (m *Module) SessionIdentity(r *http.Request) (Identity, bool) {
	for _, proxy := range m.proxies {
		if identity, ok := proxy.Identity(r); ok {
			return identity, true
		}
	}
	sess, err := m.store.Get(r, SessionName)
	if err != nil {
		return Identity{}, false
	}
	v, ok := sess.Values[SessionUserKey]
	if !ok {
		return Identity{}, false
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return Identity{}, false
	}
	return Identity{Username: s, Groups: sessionGroups(sess.Values[SessionGroupsKey])}, true
}

func (m *Module) SetSessionUser(w http.ResponseWriter, r *http.Request, username string) error {
	return m.SetSessionIdentity(w, r, Identity{Username: username})
}

func (m *Module) SetSessionIdentity(w http.ResponseWriter, r *http.Request, identity Identity) error {
	sess, _ := m.store.Get(r, SessionName)
	sess.Values[SessionUserKey] = identity.Username
	sess.Values[SessionGroupsKey] = identity.Groups
	token, err := newCSRFToken()
	if err != nil {
		return err
	}
	sess.Values[SessionCSRFKey] = token
	delete(sess.Values, RedirectURL)
	return sess.Save(r, w)
}

func (m *Module) CSRFToken(r *http.Request) (string, bool) {
	sess, err := m.store.Get(r, SessionName)
	if err != nil {
		return "", false
	}
	v, ok := sess.Values[SessionCSRFKey]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok && s != ""
}

func (m *Module) EnsureCSRFToken(w http.ResponseWriter, r *http.Request) (string, error) {
	if token, ok := m.CSRFToken(r); ok {
		return token, nil
	}
	token, err := newCSRFToken()
	if err != nil {
		return "", err
	}
	sess, _ := m.store.Get(r, SessionName)
	sess.Values[SessionCSRFKey] = token
	return token, sess.Save(r, w)
}

func (m *Module) ValidCSRF(r *http.Request) bool {
	expected, ok := m.CSRFToken(r)
	if !ok {
		return false
	}
	got := r.Header.Get("X-Cerebro-CSRF")
	return got != "" && subtleConstantTimeEqual(got, expected)
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
// When auth is enabled and there is no session, it returns 401.
func (m *Module) APIMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled {
			next.ServeHTTP(w, r)
			return
		}
		identity, ok := m.SessionIdentity(r)
		if !ok {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithIdentity(r.Context(), identity)))
	})
}

func sessionGroups(value any) []string {
	switch groups := value.(type) {
	case []string:
		return groups
	case []any:
		out := make([]string, 0, len(groups))
		for _, group := range groups {
			if s, ok := group.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func newCSRFToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func subtleConstantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
