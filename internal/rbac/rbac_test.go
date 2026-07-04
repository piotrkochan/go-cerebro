package rbac

import (
	"net/http"
	"testing"

	"github.com/lmenezes/cerebro/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthorizer_AllowsBoundRoleAndDenyOverrides(t *testing.T) {
	authorizer := New(config.RBAC{
		Enabled: true,
		Bindings: []config.RBACBinding{
			{Subject: "alice", Role: "role:editor"},
		},
		Policies: []config.RBACPolicy{
			{Subject: "role:editor", Resource: "indices", Action: "*", Object: "local-cluster/logs-*", Effect: "allow"},
			{Subject: "role:editor", Resource: "indices", Action: "delete", Object: "*", Effect: "deny"},
		},
	})

	require.NotNil(t, authorizer.enforcer)
	allowed, explanation, err := authorizer.enforcer.EnforceEx("alice", "indices", "refresh", "local-cluster/logs-000001")
	require.NoError(t, err)
	assert.True(t, allowed, explanation)
	assert.True(t, authorizer.Allow("alice", nil, Request{Resource: "indices", Action: "refresh", Object: "local-cluster/logs-000001"}))
	assert.False(t, authorizer.Allow("alice", nil, Request{Resource: "indices", Action: "delete", Object: "local-cluster/logs-000001"}))
	assert.False(t, authorizer.Allow("alice", nil, Request{Resource: "indices", Action: "refresh", Object: "local-cluster/users-000001"}))
}

func TestAuthorizer_UsesDefaultRole(t *testing.T) {
	authorizer := New(config.RBAC{
		Enabled:     true,
		DefaultRole: "role:viewer",
		Policies: []config.RBACPolicy{
			{Subject: "role:viewer", Resource: "*", Action: "read", Object: "*", Effect: "allow"},
		},
	})

	assert.True(t, authorizer.Allow("bob", nil, Request{Resource: "overview", Action: "read", Object: "local-cluster"}))
	assert.False(t, authorizer.Allow("bob", nil, Request{Resource: "indices", Action: "delete", Object: "local-cluster/logs"}))
}

func TestAuthorizer_DenyOverridesAllowAcrossPrincipals(t *testing.T) {
	authorizer := New(config.RBAC{
		Enabled:     true,
		DefaultRole: "role:viewer",
		Policies: []config.RBACPolicy{
			{Subject: "role:viewer", Resource: "*", Action: "read", Object: "*", Effect: "allow"},
			{Subject: "alice", Resource: "rest", Action: "read", Object: "*", Effect: "deny"},
		},
	})

	assert.False(t, authorizer.Allow("alice", nil, Request{Resource: "rest", Action: "read", Object: "local-cluster"}))
	assert.True(t, authorizer.Allow("bob", nil, Request{Resource: "rest", Action: "read", Object: "local-cluster"}))
}

func TestAuthorizer_UsesGroupBindings(t *testing.T) {
	authorizer := New(config.RBAC{
		Enabled: true,
		Bindings: []config.RBACBinding{
			{Subject: "group:cerebro-admins", Role: "role:admin"},
		},
		Policies: []config.RBACPolicy{
			{Subject: "role:admin", Resource: "*", Action: "*", Object: "*", Effect: "allow"},
		},
	})

	assert.True(t, authorizer.Allow("alice", []string{"cerebro-admins"}, Request{Resource: "indices", Action: "delete", Object: "prod/logs"}))
	assert.False(t, authorizer.Allow("bob", []string{"cerebro-viewers"}, Request{Resource: "indices", Action: "delete", Object: "prod/logs"}))
}

func TestAuthorizer_AllowsSystemSupportRequests(t *testing.T) {
	authorizer := New(config.RBAC{
		Enabled:  true,
		Policies: []config.RBACPolicy{{Subject: "role:viewer", Resource: "overview", Action: "read", Object: "*", Effect: "allow"}},
	})

	assert.True(t, authorizer.Allow("alice", nil, Request{Resource: "ui", Action: "read", Object: "prod", System: true}))
}

func TestAuthorizer_SupportsClusterScopedObjects(t *testing.T) {
	authorizer := New(config.RBAC{
		Enabled: true,
		Bindings: []config.RBACBinding{
			{Subject: "alice", Role: "role:prod-operator"},
		},
		Policies: []config.RBACPolicy{
			{Subject: "role:prod-operator", Resource: "indices", Action: "refresh", Object: "prod/logs-*", Effect: "allow"},
			{Subject: "role:prod-operator", Resource: "overview", Action: "read", Object: "prod", Effect: "allow"},
		},
	})

	assert.True(t, authorizer.Allow("alice", nil, Request{Resource: "indices", Action: "refresh", Object: "prod/logs-000001"}))
	assert.True(t, authorizer.Allow("alice", nil, Request{Resource: "overview", Action: "read", Object: "prod"}))
	assert.False(t, authorizer.Allow("alice", nil, Request{Resource: "indices", Action: "refresh", Object: "dev/logs-000001"}))
	assert.False(t, authorizer.Allow("alice", nil, Request{Resource: "overview", Action: "read", Object: "dev"}))
}

func TestClassifyClusterRequests(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		expected Request
	}{
		{
			name:     "overview read",
			method:   http.MethodGet,
			path:     "/clusters/local-cluster/overview",
			expected: Request{Resource: "overview", Action: "read", Object: "local-cluster"},
		},
		{
			name:     "index delete",
			method:   http.MethodDelete,
			path:     "/clusters/local-cluster/overview/indices/logs-000001",
			expected: Request{Resource: "indices", Action: "delete", Object: "local-cluster/logs-000001"},
		},
		{
			name:     "rest execute",
			method:   http.MethodPost,
			path:     "/clusters/local-cluster/rest/requests",
			expected: Request{Resource: "rest", Action: "execute", Object: "local-cluster"},
		},
		{
			name:     "data explorer save",
			method:   http.MethodPut,
			path:     "/clusters/local-cluster/data_explorer/logs-000001/documents",
			expected: Request{Resource: "documents", Action: "write", Object: "local-cluster/logs-000001"},
		},
		{
			name:     "snapshot restore",
			method:   http.MethodPost,
			path:     "/clusters/local-cluster/snapshots/fs/snap-1/restore",
			expected: Request{Resource: "snapshots", Action: "restore", Object: "local-cluster/fs/snap-1"},
		},
		{
			name:     "commons index helper maps to indices",
			method:   http.MethodGet,
			path:     "/clusters/local-cluster/commons/indices/logs-000001/settings",
			expected: Request{Resource: "indices", Action: "read", Object: "local-cluster/logs-000001"},
		},
		{
			name:     "commons nodes helper maps to nodes",
			method:   http.MethodGet,
			path:     "/clusters/local-cluster/commons/nodes/node-1/stats",
			expected: Request{Resource: "nodes", Action: "read", Object: "local-cluster/node-1"},
		},
		{
			name:     "navbar is system support",
			method:   http.MethodGet,
			path:     "/clusters/local-cluster/navbar",
			expected: Request{Resource: "ui", Action: "read", Object: "local-cluster", System: true},
		},
		{
			name:     "connect is system support",
			method:   http.MethodGet,
			path:     "/connect/hosts",
			expected: Request{Resource: "ui", Action: "read", Object: "*", System: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, Classify(tt.method, tt.path))
		})
	}
}
