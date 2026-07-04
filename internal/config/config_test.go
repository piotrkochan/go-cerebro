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
    username: "${BASIC_AUTH_USER}"
    password: "${BASIC_AUTH_PWD}"
server:
  port: 9100
  secret: "${APPLICATION_SECRET}"
`), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)
	assert.True(t, cfg.Auth.Basic.Enabled)
	assert.Equal(t, "admin", cfg.Auth.Basic.Username)
	assert.Equal(t, "s3cret", cfg.Auth.Basic.Password)
	assert.Equal(t, "from-env", cfg.Server.Secret)
	assert.True(t, cfg.Server.CSRFEnabled)
	assert.Equal(t, 9100, cfg.Server.Port)
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "text", cfg.Logging.Format)
	assert.True(t, cfg.Logging.RequestLogEnabled)
	assert.Len(t, cfg.Hosts, 1)
	assert.Equal(t, "Local", cfg.Hosts[0].Name)
	assert.Equal(t, "local", cfg.Hosts[0].ID)
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
`), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, "warn", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
	assert.False(t, cfg.Logging.RequestLogEnabled)
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
`), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)

	assert.True(t, cfg.RBAC.Enabled)
	assert.Equal(t, "role:viewer", cfg.RBAC.DefaultRole)
	assert.Equal(t, []RBACBinding{{Subject: "alice", Role: "role:admin"}}, cfg.RBAC.Bindings)
	require.Len(t, cfg.RBAC.Policies, 2)
	assert.Equal(t, "allow", cfg.RBAC.Policies[0].Effect)
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
    trusted_proxies: ["127.0.0.1/32"]
server:
  secret: "test-secret"
`), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)

	assert.True(t, cfg.Auth.Proxy.Enabled)
	assert.Equal(t, "X-Forwarded-User", cfg.Auth.Proxy.UserHeader)
	assert.Equal(t, []string{"127.0.0.1/32"}, cfg.Auth.Proxy.TrustedProxies)
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
