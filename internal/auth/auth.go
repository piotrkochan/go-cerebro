package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"github.com/lmenezes/cerebro/internal/config"
)

const (
	SessionName           = "cerebro"
	SessionUserKey        = "username"
	SessionGroupsKey      = "groups"
	SessionProviderKey    = "provider"
	SessionProviderIDKey  = "provider_id"
	SessionCSRFKey        = "csrf"
	SessionIssuedAtKey    = "issued_at"
	SessionLastSeenKey    = "last_seen"
	SessionEntraStateKey  = "entra_state"
	SessionEntraNonceKey  = "entra_nonce"
	SessionEntraReturnKey = "entra_return"
	SessionOAuthStateKey  = "oauth_state"
	SessionOAuthNonceKey  = "oauth_nonce"
	SessionOAuthReturnKey = "oauth_return"
	RedirectURL           = "redirect"
)

type Identity struct {
	Username   string
	Groups     []string
	Provider   string
	ProviderID string
}

type LoginProvider struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	LoginPath string `json:"login_path"`
	Name      string `json:"name"`
}

var ErrInvalidCredentials = errors.New("invalid credentials")

type PasswordAuthenticator interface {
	Authenticate(username, password string) (Identity, error)
}

type RequestAuthenticator interface {
	Identity(r *http.Request) (Identity, bool)
	LogoutURL() string
	ProviderID() string
}

type ExternalLoginProvider interface {
	Name() string
	ConfiguredRedirectURL() string
	AuthCodeURL(ctx context.Context, redirectURL, state, nonce string) (string, error)
	Exchange(ctx context.Context, redirectURL, code, nonce string) (Identity, error)
}

type namedPasswordAuthenticator struct {
	provider   string
	providerID string
	service    PasswordAuthenticator
}

func (s namedPasswordAuthenticator) Authenticate(username, password string) (Identity, error) {
	identity, err := s.service.Authenticate(username, password)
	if err != nil {
		return Identity{}, err
	}
	identity.Provider = s.provider
	identity.ProviderID = s.providerID
	return identity, nil
}

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
	enabled       bool
	entraID       map[string]ExternalLoginProvider
	oauth         map[string]ExternalLoginProvider
	proxies       []RequestAuthenticator
	services      []PasswordAuthenticator
	store         *sessions.CookieStore
	sessionMaxAge time.Duration
	sessionIdle   time.Duration
}

func NewModule(cfg *config.Config) (*Module, error) {
	sessionSecret := cfg.Server.Secret
	if sessionSecret == "" {
		sessionSecret = "change-me"
	}
	store := sessions.NewCookieStore([]byte(sessionSecret))
	cookieMaxAge := cfg.Auth.Session.CookieMaxAgeSeconds
	store.Options = &sessions.Options{
		Path:     strings.TrimRight(cfg.Server.BasePath, "/") + "/",
		HttpOnly: true,
		MaxAge:   cookieMaxAge,
		SameSite: http.SameSiteLaxMode,
		Secure:   cfg.Server.CookieSecure,
	}
	if store.Options.Path == "" {
		store.Options.Path = "/"
	}

	m := &Module{
		entraID:       map[string]ExternalLoginProvider{},
		oauth:         map[string]ExternalLoginProvider{},
		store:         store,
		sessionMaxAge: secondsDuration(cfg.Auth.Session.MaxLifetimeSeconds),
		sessionIdle:   secondsDuration(cfg.Auth.Session.IdleTimeoutSeconds),
	}
	for _, providerID := range providerIDs(cfg.Auth.Basic) {
		settings := cfg.Auth.Basic[providerID]
		if !settings.Enabled {
			continue
		}
		svc, err := NewBasicService(settings)
		if err != nil {
			return nil, err
		}
		m.services = append(m.services, namedPasswordAuthenticator{provider: "basic", providerID: providerID, service: svc})
	}
	for _, providerID := range providerIDs(cfg.Auth.LDAP) {
		settings := cfg.Auth.LDAP[providerID]
		if !settings.Enabled {
			continue
		}
		svc, err := NewLDAPService(settings)
		if err != nil {
			return nil, err
		}
		m.services = append(m.services, namedPasswordAuthenticator{provider: "ldap", providerID: providerID, service: svc})
	}
	for _, providerID := range providerIDs(cfg.Auth.Proxy) {
		settings := cfg.Auth.Proxy[providerID]
		if !settings.Enabled {
			continue
		}
		proxy, err := NewNamedProxyAuthenticator(providerID, settings)
		if err != nil {
			return nil, err
		}
		m.proxies = append(m.proxies, proxy)
	}
	for _, providerID := range providerIDs(cfg.Auth.EntraID) {
		settings := cfg.Auth.EntraID[providerID]
		if !settings.Enabled {
			continue
		}
		provider, err := NewEntraIDProvider(settings)
		if err != nil {
			return nil, err
		}
		m.entraID[providerID] = provider
	}
	for _, providerID := range providerIDs(cfg.Auth.OAuth) {
		settings := cfg.Auth.OAuth[providerID]
		if !settings.Enabled {
			continue
		}
		provider, err := NewOAuthProvider(settings)
		if err != nil {
			return nil, err
		}
		m.oauth[providerID] = provider
	}
	m.enabled = len(m.services) > 0 || len(m.proxies) > 0 || len(m.entraID) > 0 || len(m.oauth) > 0
	return m, nil
}

func (m *Module) Enabled() bool { return m.enabled }

func (m *Module) PasswordLoginEnabled() bool { return len(m.services) > 0 }

func (m *Module) EntraIDEnabled() bool { return len(m.entraID) > 0 }

func (m *Module) OAuthEnabled() bool { return len(m.oauth) > 0 }

func (m *Module) OAuthName() string {
	keys := sortedMapKeys(m.oauth)
	if len(keys) == 0 {
		return ""
	}
	return m.oauth[keys[0]].Name()
}

func (m *Module) ExternalLoginProviders() []LoginProvider {
	providers := make([]LoginProvider, 0, len(m.entraID)+len(m.oauth))
	for _, providerID := range sortedMapKeys(m.entraID) {
		provider := m.entraID[providerID]
		providers = append(providers, LoginProvider{
			ID:        providerID,
			Kind:      "entra_id",
			LoginPath: "/auth/entraid/" + providerID + "/login",
			Name:      provider.Name(),
		})
	}
	for _, providerID := range sortedMapKeys(m.oauth) {
		provider := m.oauth[providerID]
		providers = append(providers, LoginProvider{
			ID:        providerID,
			Kind:      "oauth",
			LoginPath: "/auth/oauth/" + providerID + "/login",
			Name:      provider.Name(),
		})
	}
	return providers
}

func (m *Module) EntraIDRedirectURL(providerID string) string {
	provider, ok := m.entraID[providerID]
	if !ok {
		return ""
	}
	return provider.ConfiguredRedirectURL()
}

func (m *Module) OAuthRedirectURL(providerID string) string {
	provider, ok := m.oauth[providerID]
	if !ok {
		return ""
	}
	return provider.ConfiguredRedirectURL()
}

func (m *Module) LogoutRedirectURL(identity Identity, fallback string) string {
	if identity.Provider == "proxy" {
		for _, proxy := range m.proxies {
			if proxy.ProviderID() == identity.ProviderID {
				if logoutURL := proxy.LogoutURL(); logoutURL != "" {
					return logoutURL
				}
			}
		}
	}
	return fallback
}

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
	if m.sessionExpired(sess) {
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
	provider, _ := sess.Values[SessionProviderKey].(string)
	providerID, _ := sess.Values[SessionProviderIDKey].(string)
	return Identity{Username: s, Groups: sessionGroups(sess.Values[SessionGroupsKey]), Provider: provider, ProviderID: providerID}, true
}

func (m *Module) SetSessionUser(w http.ResponseWriter, r *http.Request, username string) error {
	return m.SetSessionIdentity(w, r, Identity{Username: username})
}

func (m *Module) SetSessionIdentity(w http.ResponseWriter, r *http.Request, identity Identity) error {
	sess, _ := m.store.Get(r, SessionName)
	return m.saveSessionIdentity(w, r, sess, identity)
}

func (m *Module) saveSessionIdentity(w http.ResponseWriter, r *http.Request, sess *sessions.Session, identity Identity) error {
	redirect, _ := sess.Values[RedirectURL].(string)
	resetSession(sess)
	sess.Values[SessionUserKey] = identity.Username
	sess.Values[SessionGroupsKey] = identity.Groups
	sess.Values[SessionProviderKey] = identity.Provider
	sess.Values[SessionProviderIDKey] = identity.ProviderID
	if safe := safeRedirect(redirect); safe != "" {
		sess.Values[RedirectURL] = safe
	}
	now := time.Now().Unix()
	sess.Values[SessionIssuedAtKey] = now
	sess.Values[SessionLastSeenKey] = now
	token, err := newCSRFToken()
	if err != nil {
		return err
	}
	sess.Values[SessionCSRFKey] = token
	return sess.Save(r, w)
}

func (m *Module) TouchSession(w http.ResponseWriter, r *http.Request) error {
	if m.sessionIdle <= 0 {
		return nil
	}
	sess, err := m.store.Get(r, SessionName)
	if err != nil || m.sessionExpired(sess) {
		return nil
	}
	if _, ok := sess.Values[SessionUserKey].(string); !ok {
		return nil
	}
	sess.Values[SessionLastSeenKey] = time.Now().Unix()
	return sess.Save(r, w)
}

func (m *Module) BeginEntraIDLogin(providerID string, w http.ResponseWriter, r *http.Request, callbackURL, returnPath string) (string, error) {
	provider, ok := m.entraID[providerID]
	if !ok {
		return "", errors.New("entra id authentication not enabled")
	}
	state, err := newCSRFToken()
	if err != nil {
		return "", err
	}
	nonce, err := newCSRFToken()
	if err != nil {
		return "", err
	}
	sess, _ := m.store.Get(r, SessionName)
	sess.Values[sessionFlowKey(SessionEntraStateKey, providerID)] = state
	sess.Values[sessionFlowKey(SessionEntraNonceKey, providerID)] = nonce
	if safe := safeRedirect(returnPath); safe != "" {
		sess.Values[sessionFlowKey(SessionEntraReturnKey, providerID)] = safe
	} else {
		delete(sess.Values, sessionFlowKey(SessionEntraReturnKey, providerID))
	}
	if err := sess.Save(r, w); err != nil {
		return "", err
	}
	return provider.AuthCodeURL(r.Context(), callbackURL, state, nonce)
}

func (m *Module) CompleteEntraIDLogin(providerID string, w http.ResponseWriter, r *http.Request, callbackURL string) (string, error) {
	provider, ok := m.entraID[providerID]
	if !ok {
		return "", errors.New("entra id authentication not enabled")
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		return "", ErrInvalidCredentials
	}
	sess, err := m.store.Get(r, SessionName)
	if err != nil {
		return "", err
	}
	state, ok := sess.Values[sessionFlowKey(SessionEntraStateKey, providerID)].(string)
	if !ok || state == "" || !subtleConstantTimeEqual(r.URL.Query().Get("state"), state) {
		return "", ErrInvalidCredentials
	}
	nonce, ok := sess.Values[sessionFlowKey(SessionEntraNonceKey, providerID)].(string)
	if !ok || nonce == "" {
		return "", ErrInvalidCredentials
	}
	returnPath, _ := sess.Values[sessionFlowKey(SessionEntraReturnKey, providerID)].(string)
	identity, err := provider.Exchange(r.Context(), callbackURL, code, nonce)
	if err != nil {
		return "", err
	}
	identity.Provider = "entra_id"
	identity.ProviderID = providerID
	if err := m.saveSessionIdentity(w, r, sess, identity); err != nil {
		return "", err
	}
	return safeRedirect(returnPath), nil
}

func (m *Module) BeginOAuthLogin(providerID string, w http.ResponseWriter, r *http.Request, callbackURL, returnPath string) (string, error) {
	provider, ok := m.oauth[providerID]
	if !ok {
		return "", errors.New("oauth authentication not enabled")
	}
	state, err := newCSRFToken()
	if err != nil {
		return "", err
	}
	nonce, err := newCSRFToken()
	if err != nil {
		return "", err
	}
	sess, _ := m.store.Get(r, SessionName)
	sess.Values[sessionFlowKey(SessionOAuthStateKey, providerID)] = state
	sess.Values[sessionFlowKey(SessionOAuthNonceKey, providerID)] = nonce
	if safe := safeRedirect(returnPath); safe != "" {
		sess.Values[sessionFlowKey(SessionOAuthReturnKey, providerID)] = safe
	} else {
		delete(sess.Values, sessionFlowKey(SessionOAuthReturnKey, providerID))
	}
	if err := sess.Save(r, w); err != nil {
		return "", err
	}
	return provider.AuthCodeURL(r.Context(), callbackURL, state, nonce)
}

func (m *Module) CompleteOAuthLogin(providerID string, w http.ResponseWriter, r *http.Request, callbackURL string) (string, error) {
	provider, ok := m.oauth[providerID]
	if !ok {
		return "", errors.New("oauth authentication not enabled")
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		return "", ErrInvalidCredentials
	}
	sess, err := m.store.Get(r, SessionName)
	if err != nil {
		return "", err
	}
	state, ok := sess.Values[sessionFlowKey(SessionOAuthStateKey, providerID)].(string)
	if !ok || state == "" || !subtleConstantTimeEqual(r.URL.Query().Get("state"), state) {
		return "", ErrInvalidCredentials
	}
	nonce, ok := sess.Values[sessionFlowKey(SessionOAuthNonceKey, providerID)].(string)
	if !ok || nonce == "" {
		return "", ErrInvalidCredentials
	}
	returnPath, _ := sess.Values[sessionFlowKey(SessionOAuthReturnKey, providerID)].(string)
	identity, err := provider.Exchange(r.Context(), callbackURL, code, nonce)
	if err != nil {
		return "", err
	}
	identity.Provider = "oauth"
	identity.ProviderID = providerID
	if err := m.saveSessionIdentity(w, r, sess, identity); err != nil {
		return "", err
	}
	return safeRedirect(returnPath), nil
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

func (m *Module) ClearSessionCookies(r *http.Request) ([]string, error) {
	w := &headerOnlyResponseWriter{header: http.Header{}}
	if err := m.ClearSession(w, r); err != nil {
		return nil, err
	}
	return w.header.Values("Set-Cookie"), nil
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
		_ = m.TouchSession(w, r)
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

func (m *Module) sessionExpired(sess *sessions.Session) bool {
	now := time.Now()
	if m.sessionMaxAge > 0 {
		issuedAt, ok := sessionUnix(sess.Values[SessionIssuedAtKey])
		if !ok || now.Sub(issuedAt) > m.sessionMaxAge {
			return true
		}
	}
	if m.sessionIdle > 0 {
		lastSeen, ok := sessionUnix(sess.Values[SessionLastSeenKey])
		if !ok || now.Sub(lastSeen) > m.sessionIdle {
			return true
		}
	}
	return false
}

func sessionUnix(value any) (time.Time, bool) {
	switch typed := value.(type) {
	case int64:
		return time.Unix(typed, 0), true
	case int:
		return time.Unix(int64(typed), 0), true
	case float64:
		return time.Unix(int64(typed), 0), true
	default:
		return time.Time{}, false
	}
}

func resetSession(sess *sessions.Session) {
	for key := range sess.Values {
		delete(sess.Values, key)
	}
}

type headerOnlyResponseWriter struct {
	header http.Header
}

func (w *headerOnlyResponseWriter) Header() http.Header {
	return w.header
}

func (w *headerOnlyResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (w *headerOnlyResponseWriter) WriteHeader(int) {}

func secondsDuration(seconds int) time.Duration {
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func providerIDs[T any](providers map[string]T) []string {
	return sortedMapKeys(providers)
}

func sortedMapKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sessionFlowKey(base, providerID string) string {
	return base + ":" + providerID
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
