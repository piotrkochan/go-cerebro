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
	t.Setenv("AUTH_TYPE", "basic")
	t.Setenv("BASIC_AUTH_USER", "admin")
	t.Setenv("BASIC_AUTH_PWD", "s3cret")

	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
hosts:
  - name: "Local"
    host: "http://localhost:9200"
auth:
  type: "${AUTH_TYPE}"
  settings:
    username: "${BASIC_AUTH_USER}"
    password: "${BASIC_AUTH_PWD}"
server:
  port: 9100
  secret: "${APPLICATION_SECRET}"
`), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "basic", cfg.Auth.Type)
	assert.Equal(t, "admin", cfg.Auth.Settings.Username)
	assert.Equal(t, "s3cret", cfg.Auth.Settings.Password)
	assert.Equal(t, "from-env", cfg.Server.Secret)
	assert.Equal(t, 9100, cfg.Server.Port)
	assert.Len(t, cfg.Hosts, 1)
	assert.Equal(t, "Local", cfg.Hosts[0].Name)
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
