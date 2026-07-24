//go:build e2e

package e2e

import (
	"os"
	"strings"
	"testing"

	"github.com/lmenezes/cerebro/internal/auth"
	"github.com/lmenezes/cerebro/internal/config"
	"github.com/lmenezes/cerebro/internal/rbac"
	"github.com/stretchr/testify/require"
)

func TestLDAPRBACGroupsOverLDAPS(t *testing.T) {
	ldapURL := strings.TrimSpace(os.Getenv("CEREBRO_E2E_LDAP_URL"))
	caCertFile := strings.TrimSpace(os.Getenv("CEREBRO_E2E_LDAP_CA_CERT_FILE"))
	if ldapURL == "" || caCertFile == "" {
		t.Skip("set CEREBRO_E2E_LDAP_URL and CEREBRO_E2E_LDAP_CA_CERT_FILE to run LDAP RBAC e2e tests")
	}

	service, err := auth.NewLDAPService(config.LDAPAuth{
		URL:          ldapURL,
		CACertFile:   caCertFile,
		BaseDN:       "dc=planetexpress,dc=com",
		UserTemplate: "cn=%s,ou=people,%s",
		BindDN:       "cn=admin,dc=planetexpress,dc=com",
		BindPW:       "GoodNewsEveryone",
		GroupSearch: &config.GroupSearch{
			BaseDN:           "ou=people,dc=planetexpress,dc=com",
			UserAttr:         "member",
			UserAttrTemplate: "cn=%s,ou=people,%s",
			Group:            "(objectClass=Group)",
			NameAttr:         "cn",
		},
	})
	require.NoError(t, err)

	admin, err := service.Authenticate("Hubert J. Farnsworth", "professor")
	require.NoError(t, err)
	require.Equal(t, "Hubert J. Farnsworth", admin.Username)
	require.Contains(t, admin.Groups, "admin_staff")
	require.Contains(t, admin.Groups, "cn=admin_staff,ou=people,dc=planetexpress,dc=com")

	fry, err := service.Authenticate("Philip J. Fry", "fry")
	require.NoError(t, err)
	require.Contains(t, fry.Groups, "ship_crew")
	require.NotContains(t, fry.Groups, "admin_staff")

	authorizer := rbac.New(config.RBAC{
		Enabled: true,
		Bindings: []config.RBACBinding{
			{Subject: "group:admin_staff", Role: "role:admin"},
		},
		Policies: []config.RBACPolicy{
			{Subject: "role:admin", Resource: "indices", Action: "delete", Object: "prod/logs-*", Effect: "allow"},
		},
	})

	req := rbac.Request{Resource: "indices", Action: "delete", Object: "prod/logs-000001"}
	require.True(t, authorizer.Allow(admin.Username, admin.Groups, req))
	require.False(t, authorizer.Allow(fry.Username, fry.Groups, req))
}
