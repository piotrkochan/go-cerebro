package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_DefaultsAndEnvExpand(t *testing.T) {
	t.Setenv("APPLICATION_SECRET", "from-env")
	t.Setenv("BASIC_AUTH_USER", "admin")
	t.Setenv("BASIC_AUTH_PWD", "s3cret")

	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
hosts:
  - name: "Local"
    id: "local"
    host: "http://localhost:9200"
auth:
  basic:
    enabled: true
    users:
      - username: "${BASIC_AUTH_USER}"
        password: "${BASIC_AUTH_PWD}"
server:
  port: 9100
  secret: "${APPLICATION_SECRET}"
`), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)
	basic := cfg.Auth.Basic["basic"]
	assert.True(t, basic.Enabled)
	require.Len(t, basic.Users, 1)
	assert.Equal(t, "admin", basic.Users[0].Username)
	assert.Equal(t, "s3cret", basic.Users[0].Password)
	assert.Equal(t, "from-env", cfg.Server.Secret)
	assert.True(t, cfg.Server.CSRFEnabled)
	assert.Equal(t, 9100, cfg.Server.Port)
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "text", cfg.Logging.Format)
	assert.True(t, cfg.Logging.RequestLogEnabled)
	assert.True(t, cfg.Logging.AuthLogEnabled)
	assert.Len(t, cfg.Hosts, 1)
	assert.Equal(t, "Local", cfg.Hosts[0].Name)
	assert.Equal(t, "local", cfg.Hosts[0].ID)
}

func TestLoad_DevConfigUsesDockerComposeAuthEnv(t *testing.T) {
	t.Setenv("APPLICATION_SECRET", "docker-compose-dev-auth-secret-change-me")
	t.Setenv("BASIC_AUTH_ENABLED", "true")
	t.Setenv("BASIC_AUTH_USER", "admin")
	t.Setenv("BASIC_AUTH_PWD", "admin123")

	cfg, err := Load("../../conf/application.dev.yaml")
	require.NoError(t, err)

	basic := cfg.Auth.Basic["basic"]
	assert.True(t, basic.Enabled)
	require.Len(t, basic.Users, 2)
	assert.Equal(t, "admin", basic.Users[0].Username)
	assert.Equal(t, "admin123", basic.Users[0].Password)
	assert.Equal(t, []string{"cerebro-admins"}, basic.Users[0].Groups)
	assert.Equal(t, "user", basic.Users[1].Username)
	assert.Equal(t, "user123", basic.Users[1].Password)
	assert.Equal(t, []string{"cerebro-viewers"}, basic.Users[1].Groups)
	assert.Equal(t, "docker-compose-dev-auth-secret-change-me", cfg.Server.Secret)
	assert.True(t, cfg.RBAC.Enabled)
	assert.Empty(t, cfg.RBAC.DefaultRole)
	assert.Contains(t, cfg.RBAC.Bindings, RBACBinding{Subject: "group:cerebro-viewers", Role: "role:viewer"})
	assert.Contains(t, cfg.RBAC.Policies, RBACPolicy{Subject: "role:viewer", Resource: "*", Action: "read", Object: "*", Effect: "allow"})
}

func TestLoad_NamedAuthProviderConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
auth:
  basic:
    local_dev:
      enabled: true
      users:
        - username: "admin"
          password: "admin123"
    backup-1:
      enabled: false
server:
  secret: "test-secret"
`), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)

	require.Contains(t, cfg.Auth.Basic, "local_dev")
	require.Contains(t, cfg.Auth.Basic, "backup-1")
	assert.True(t, cfg.Auth.Basic["local_dev"].Enabled)
	assert.False(t, cfg.Auth.Basic["backup-1"].Enabled)
}

func TestLoad_RejectsMixedInlineAndNamedAuthProviderConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
auth:
  basic:
    enabled: true
    local_dev:
      enabled: true
server:
  secret: "test-secret"
`), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mixes inline provider settings and named providers")
}

func TestLoad_RejectsDuplicateEnabledAuthProviderIDs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
auth:
  basic:
    local:
      enabled: true
      users:
        - username: "admin"
          password: "admin123"
  proxy:
    local:
      enabled: true
      user_header: "X-Forwarded-User"
      trusted_proxies: ["127.0.0.1/32"]
server:
  secret: "test-secret"
`), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `auth provider id "local" is duplicated`)
}

func TestLoad_RejectsDuplicateDisabledAuthProviderIDs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
auth:
  basic:
    local:
      enabled: false
  proxy:
    local:
      enabled: false
`), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `auth provider id "local" is duplicated`)
}

func TestLoad_AllowsMultipleInlineAuthProviders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
auth:
  basic:
    enabled: true
    users:
      - username: "admin"
        password: "admin123"
  proxy:
    enabled: true
    user_header: "X-Forwarded-User"
    trusted_proxies: ["127.0.0.1/32"]
server:
  secret: "test-secret"
`), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)

	require.Contains(t, cfg.Auth.Basic, "basic")
	require.Contains(t, cfg.Auth.Proxy, "proxy")
	assert.True(t, cfg.Auth.Basic["basic"].Enabled)
	assert.True(t, cfg.Auth.Proxy["proxy"].Enabled)
}

func TestLoad_AllowsDisablingCSRF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
server:
  csrf_enabled: false
`), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)

	assert.False(t, cfg.Server.CSRFEnabled)
}

func TestLoad_LoggingConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
logging:
  level: "WARN"
  format: "json"
  request_log_enabled: false
  auth_log_enabled: false
`), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, "warn", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
	assert.False(t, cfg.Logging.RequestLogEnabled)
	assert.False(t, cfg.Logging.AuthLogEnabled)
}

func TestLoad_RBACConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
rbac:
  enabled: true
  default_role: "role:viewer"
  bindings:
    - subject: "alice"
      role: "role:admin"
  policies:
    - subject: "role:admin"
      resource: "*"
      action: "*"
      object: "*"
      effect: "allow"
    - subject: "role:viewer"
      resource: "*"
      action: "read"
      object: "*"
      effect: "allow"
auth:
  basic:
    enabled: true
    users:
      - username: "admin"
        password: "admin123"
server:
  secret: "test-secret"
`), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)

	assert.True(t, cfg.RBAC.Enabled)
	assert.Equal(t, "role:viewer", cfg.RBAC.DefaultRole)
	assert.Equal(t, []RBACBinding{{Subject: "alice", Role: "role:admin"}}, cfg.RBAC.Bindings)
	require.Len(t, cfg.RBAC.Policies, 2)
	assert.Equal(t, "allow", cfg.RBAC.Policies[0].Effect)
}

func TestLoad_RBACDefaultRoleIsOptional(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
rbac:
  enabled: true
  bindings:
    - {subject: "alice", role: "role:admin"}
  policies:
    - {subject: "role:admin", resource: "*", action: "*", object: "*", effect: "allow"}
auth:
  basic:
    enabled: true
    users:
      - username: "admin"
        password: "admin123"
server:
  secret: "test-secret"
`), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)

	assert.True(t, cfg.RBAC.Enabled)
	assert.Empty(t, cfg.RBAC.DefaultRole)
	assert.Equal(t, []RBACBinding{{Subject: "alice", Role: "role:admin"}}, cfg.RBAC.Bindings)
	require.Len(t, cfg.RBAC.Policies, 1)
}

func TestLoad_RBACAllowsIndexFlushAction(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
auth:
  basic:
    enabled: true
    users:
      - username: "admin"
        password: "admin123"
server:
  secret: "test-secret"
rbac:
  enabled: true
  policies:
    - {subject: "admin", resource: "indices", action: "flush", object: "*", effect: "allow"}
`), 0o600))

	cfg, err := Load(path)

	require.NoError(t, err)
	require.Len(t, cfg.RBAC.Policies, 1)
	assert.Equal(t, "flush", cfg.RBAC.Policies[0].Action)
}

func TestLoad_RBACRequiresAuthProvider(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
rbac:
  enabled: true
  policies:
    - {subject: "role:viewer", resource: "*", action: "read", object: "*", effect: "allow"}
`), 0o600))

	_, err := Load(path)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "rbac.enabled requires at least one auth provider")
}

func TestLoad_ServerPublicURLAndTrustedProxies(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
server:
  public_url: "https://cerebro.example.org"
  trusted_proxies: ["127.0.0.1/32", "10.0.0.10"]
`), 0o600))

	cfg, err := Load(path)

	require.NoError(t, err)
	assert.Equal(t, "https://cerebro.example.org", cfg.Server.PublicURL)
	assert.Equal(t, []string{"127.0.0.1/32", "10.0.0.10"}, cfg.Server.TrustedProxies)
}

func TestLoad_RejectsInvalidServerPublicURLAndTrustedProxy(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "public url", body: "public_url: \"://bad\"", want: "server.public_url"},
		{name: "trusted proxy", body: "trusted_proxies: [\"bad\"]", want: "server.trusted_proxies"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "app.yaml")
			require.NoError(t, os.WriteFile(path, []byte("server:\n  "+tt.body+"\n"), 0o600))

			_, err := Load(path)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestLoad_ProxyAuthConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
auth:
  proxy:
    enabled: true
    user_header: "X-Forwarded-User"
    groups_header: "X-Forwarded-Groups"
    default_groups: ["cerebro-viewers"]
    logout_url: "/oauth2/sign_out?rd=/oauth2/sign_in"
    trusted_proxies: ["127.0.0.1/32"]
server:
  secret: "test-secret"
`), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)

	proxy := cfg.Auth.Proxy["proxy"]
	assert.True(t, proxy.Enabled)
	assert.Equal(t, "X-Forwarded-User", proxy.UserHeader)
	assert.Equal(t, []string{"cerebro-viewers"}, proxy.DefaultGroups)
	assert.Equal(t, "/oauth2/sign_out?rd=/oauth2/sign_in", proxy.LogoutURL)
	assert.Equal(t, []string{"127.0.0.1/32"}, proxy.TrustedProxies)
}

func TestLoad_RejectsUnsafeProxyLogoutURL(t *testing.T) {
	tests := []string{"javascript:alert(1)", "//evil.example.org/logout", `/\evil`}
	for _, logoutURL := range tests {
		t.Run(logoutURL, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "app.yaml")
			require.NoError(t, os.WriteFile(path, []byte(`
auth:
  proxy:
    enabled: true
    user_header: "X-Forwarded-User"
    logout_url: '`+logoutURL+`'
    trusted_proxies: ["127.0.0.1/32"]
server:
  secret: "test-secret"
`), 0o600))

			_, err := Load(path)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "auth.proxy.proxy.logout_url")
		})
	}
}

func TestLoad_AuthSessionConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
auth:
  session:
    cookie_max_age_seconds: 28800
    max_lifetime_seconds: 28800
    idle_timeout_seconds: 1800
`), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, 28800, cfg.Auth.Session.CookieMaxAgeSeconds)
	assert.Equal(t, 28800, cfg.Auth.Session.MaxLifetimeSeconds)
	assert.Equal(t, 1800, cfg.Auth.Session.IdleTimeoutSeconds)
}

func TestLoad_RejectsNegativeAuthSessionTimeouts(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "cookie max age", body: "cookie_max_age_seconds: -1"},
		{name: "max lifetime", body: "max_lifetime_seconds: -1"},
		{name: "idle timeout", body: "idle_timeout_seconds: -1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "app.yaml")
			require.NoError(t, os.WriteFile(path, []byte("auth:\n  session:\n    "+tt.body+"\n"), 0o600))

			_, err := Load(path)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "auth.session")
		})
	}
}

func TestLoad_EntraIDAuthConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
auth:
  entra_id:
    enabled: true
    tenant_id: "example.onmicrosoft.com"
    client_id: "client-id"
    client_secret: "client-secret"
    redirect_url: "https://cerebro.example.org/auth/entraid/callback"
    username_claim: "email"
    groups_claim: "roles"
    default_groups: ["cerebro-viewers"]
server:
  secret: "test-secret"
`), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)

	entraID := cfg.Auth.EntraID["entra_id"]
	assert.True(t, entraID.Enabled)
	assert.Equal(t, "example.onmicrosoft.com", entraID.TenantID)
	assert.Equal(t, "client-id", entraID.ClientID)
	assert.Equal(t, "roles", entraID.GroupsClaim)
	assert.Equal(t, []string{"cerebro-viewers"}, entraID.DefaultGroups)
}

func TestLoad_OAuthAuthConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
auth:
  oauth:
    enabled: true
    name: "GitHub"
    auth_url: "https://github.com/login/oauth/authorize"
    token_url: "https://github.com/login/oauth/access_token"
    userinfo_url: "https://api.github.com/user"
    client_id: "client-id"
    client_secret: "client-secret"
    redirect_url: "https://cerebro.example.org/auth/oauth/callback"
    username_claim: "login"
    groups_claim: "teams"
    default_groups: ["cerebro-viewers"]
    scopes: ["read:user"]
server:
  secret: "test-secret"
`), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)

	oauth := cfg.Auth.OAuth["oauth"]
	assert.True(t, oauth.Enabled)
	assert.Equal(t, "GitHub", oauth.Name)
	assert.Equal(t, "https://github.com/login/oauth/authorize", oauth.AuthURL)
	assert.Equal(t, "login", oauth.UsernameClaim)
	assert.Equal(t, []string{"cerebro-viewers"}, oauth.DefaultGroups)
	assert.Equal(t, []string{"read:user"}, oauth.Scopes)
}

func TestLoad_OAuthOIDCConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
auth:
  oauth:
    enabled: true
    name: "Dex"
    issuer_url: "https://dex.example.org"
    client_id: "client-id"
    client_secret: "client-secret"
server:
  secret: "test-secret"
`), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)

	oauth := cfg.Auth.OAuth["oauth"]
	assert.True(t, oauth.Enabled)
	assert.Equal(t, "Dex", oauth.Name)
	assert.Equal(t, "https://dex.example.org", oauth.IssuerURL)
}

func TestLoad_LDAPRequiredGroupsConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
auth:
  ldap:
    enabled: true
    url: "ldaps://ldap.example.org:636"
    base_dn: "dc=example,dc=org"
    user_template: "uid=%s,%s"
    default_groups: ["cerebro-viewers"]
    required_groups: ["cerebro-admins"]
server:
  secret: "test-secret"
`), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, []string{"cerebro-viewers"}, cfg.Auth.LDAP["ldap"].DefaultGroups)
	assert.Equal(t, []string{"cerebro-admins"}, cfg.Auth.LDAP["ldap"].RequiredGroups)
}

func TestLoad_RejectsIncompleteOAuthAuth(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
auth:
  oauth:
    enabled: true
    auth_url: "https://auth.example.org/authorize"
    token_url: "https://auth.example.org/token"
    client_id: "client-id"
    client_secret: "client-secret"
server:
  secret: "test-secret"
`), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth.oauth.oauth.userinfo_url")
}

func TestLoad_RejectsInsecureOAuthEndpoint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
auth:
  oauth:
    enabled: true
    auth_url: "http://auth.example.org/authorize"
    token_url: "https://auth.example.org/token"
    userinfo_url: "https://auth.example.org/userinfo"
    client_id: "client-id"
    client_secret: "client-secret"
server:
  secret: "test-secret"
`), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth.oauth.oauth.auth_url")
}

func TestLoad_RejectsIncompleteEntraIDAuth(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
auth:
  entra_id:
    enabled: true
    tenant_id: "example.onmicrosoft.com"
    client_id: "client-id"
server:
  secret: "test-secret"
`), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth.entra_id.entra_id.client_secret")
}

func TestLoad_ValidatesEntraIDIssuerURL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
auth:
  entra_id:
    enabled: true
    issuer_url: "http://example.org"
    client_id: "client-id"
    client_secret: "client-secret"
server:
  secret: "test-secret"
`), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth.entra_id.entra_id.issuer_url")
}

func TestLoad_RejectsProxyAuthWithoutTrustedProxies(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
auth:
  proxy:
    enabled: true
    user_header: "X-Forwarded-User"
server:
  secret: "test-secret"
`), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "trusted_proxies")
}

func TestLoad_RejectsInvalidRBACPolicy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
auth:
  basic:
    enabled: true
    users:
      - username: "admin"
        password: "admin123"
server:
  secret: "test-secret"
rbac:
  enabled: true
  policies:
    - subject: "role:admin"
      resource: "*"
      action: "*"
      object: "*"
      effect: "maybe"
`), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "effect")
}

func TestLoad_RejectsUnsupportedRBACResource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
auth:
  basic:
    enabled: true
    users:
      - username: "admin"
        password: "admin123"
server:
  secret: "test-secret"
rbac:
  enabled: true
  policies:
    - {subject: "role:admin", resource: "commons", action: "read", object: "*", effect: "allow"}
`), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported resource")
}

func TestLoad_RejectsUnsupportedRBACAction(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
auth:
  basic:
    enabled: true
    users:
      - username: "admin"
        password: "admin123"
server:
  secret: "test-secret"
rbac:
  enabled: true
  policies:
    - {subject: "role:admin", resource: "nodes", action: "delete", object: "*", effect: "allow"}
`), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported action")
}

func TestLoad_RejectsInvalidLoggingLevel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
logging:
  level: "trace"
`), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "logging.level")
}

func TestLoad_RejectsInvalidLoggingFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
logging:
  format: "pretty"
`), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "logging.format")
}

func TestLoad_HostByName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
hosts:
  - name: "Prod"
    host: "https://prod:9200"
    auth: {username: u, password: p}
    headers_whitelist: ["X-Forwarded-For"]
`), 0o600))
	cfg, err := Load(path)
	require.NoError(t, err)
	h, ok := cfg.HostByName("Prod")
	require.True(t, ok)
	require.NotNil(t, h.Auth)
	assert.Equal(t, "u", h.Auth.Username)
	assert.Contains(t, h.HeadersWhitelist, "X-Forwarded-For")
	h, ok = cfg.HostByName("https://prod:9200")
	require.True(t, ok)
	assert.Equal(t, "Prod", h.Name)
	_, ok = cfg.HostByName("Missing")
	assert.False(t, ok)
}

func TestHostRefs_UsesStableUniqueSlugs(t *testing.T) {
	cfg := &Config{Hosts: []Host{
		{Name: "Moj klaster 01", Host: "http://one:9200"},
		{Name: "Moj-klaster 01", Host: "http://two:9200"},
		{ID: "prod-main", Name: "Prod", Host: "http://prod:9200"},
	}}

	assert.Equal(t, []HostRef{
		{Name: "Moj klaster 01", Slug: "moj-klaster-01"},
		{Name: "Moj-klaster 01", Slug: "moj-klaster-01-2"},
		{Name: "Prod", Slug: "prod-main"},
	}, cfg.HostRefs())

	host, ok := cfg.HostBySlug("moj-klaster-01-2")
	require.True(t, ok)
	assert.Equal(t, "http://two:9200", host.Host)
	host, ok = cfg.HostBySlug("prod-main")
	require.True(t, ok)
	assert.Equal(t, "http://prod:9200", host.Host)
}

func TestLoad_RejectsInvalidHostID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
hosts:
  - name: "Prod"
    id: "Prod_01"
    host: "https://prod:9200"
`), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid host")
	assert.Contains(t, err.Error(), "id")
}

func TestLoad_RejectsDuplicateHostID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
hosts:
  - name: "Prod one"
    id: "prod"
    host: "https://prod-one:9200"
  - name: "Prod two"
    id: "prod"
    host: "https://prod-two:9200"
`), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate host id")
}

func TestLoad_RequiresESClientCertAndKeyTogether(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
es:
  client_cert_file: "/tmp/client.pem"
`), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client_cert_file")
}

func TestLoad_RejectsCredentialsInHostURL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
hosts:
  - host: "https://elastic:secret@example.com:9200"
`), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "credentials in host URL are not allowed")
}

func TestLoad_RequiresServerTLSCertAndKeyTogether(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
server:
  tls_cert_file: "/tmp/tls.crt"
`), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server.tls_cert_file")
}

func TestLoad_RequiresAWSRegionWhenSigningEnabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
es:
  aws:
    enabled: true
`), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "es.aws.region")
}

func TestLoad_RequiresAWSStaticCredentialsTogether(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
es:
  aws:
    enabled: true
    region: eu-central-1
    access_key_id: key
`), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "es.aws.access_key_id")
}

func TestLoad_RequiresBasicAuthUsers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
auth:
  basic:
    enabled: true
server:
  secret: "test-secret"
`), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth.basic.basic.users")
}

func TestLoad_RejectsDuplicateBasicAuthUsers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
auth:
  basic:
    enabled: true
    users:
      - username: "admin"
        password: "admin123"
      - username: "admin"
        password: "other"
server:
  secret: "test-secret"
`), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicated")
}
