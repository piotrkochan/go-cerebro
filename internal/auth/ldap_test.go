package auth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lmenezes/cerebro/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLDAPService_RequiresLDAPSUnlessExplicitlyInsecure(t *testing.T) {
	_, err := NewLDAPService(config.AuthSettings{
		URL:          "ldap://ldap.example:389",
		BaseDN:       "dc=example,dc=org",
		UserTemplate: "uid=%s,%s",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires ldaps")
}

func TestNewLDAPService_RejectsInvalidCAFile(t *testing.T) {
	dir := t.TempDir()
	caFile := filepath.Join(dir, "ldap-ca.pem")
	require.NoError(t, os.WriteFile(caFile, []byte("not a pem"), 0o600))

	_, err := NewLDAPService(config.AuthSettings{
		URL:          "ldaps://ldap.example:636",
		CACertFile:   caFile,
		BaseDN:       "dc=example,dc=org",
		UserTemplate: "uid=%s,%s",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "valid PEM certificate")
}
