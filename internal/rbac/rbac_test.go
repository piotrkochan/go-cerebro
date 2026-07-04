package rbac

import (
	"net/http"
	"strings"
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

func TestAuthorizer_DefaultRoleIsOptional(t *testing.T) {
	authorizer := New(config.RBAC{
		Enabled: true,
		Policies: []config.RBACPolicy{
			{Subject: "role:viewer", Resource: "*", Action: "read", Object: "*", Effect: "allow"},
			{Subject: "alice", Resource: "overview", Action: "read", Object: "prod", Effect: "allow"},
		},
	})

	assert.True(t, authorizer.Allow("alice", nil, Request{Resource: "overview", Action: "read", Object: "prod"}))
	assert.False(t, authorizer.Allow("bob", nil, Request{Resource: "overview", Action: "read", Object: "prod"}))
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

func TestAuthorizer_MatchesDocumentedObjectPatterns(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		request  Request
		expected bool
	}{
		{
			name:     "index prefix matches prod index object",
			pattern:  "prod/index-*",
			request:  Request{Resource: "indices", Action: "read", Object: "prod/index-users"},
			expected: true,
		},
		{
			name:     "index prefix does not match different prefix",
			pattern:  "prod/index-*",
			request:  Request{Resource: "indices", Action: "read", Object: "prod/app-users"},
			expected: false,
		},
		{
			name:     "index prefix does not cross slash",
			pattern:  "prod/index-*",
			request:  Request{Resource: "indices", Action: "read", Object: "prod/index-users/settings"},
			expected: false,
		},
		{
			name:     "one level cluster wildcard matches index object",
			pattern:  "prod/*",
			request:  Request{Resource: "indices", Action: "read", Object: "prod/index-users"},
			expected: true,
		},
		{
			name:     "one level cluster wildcard does not match snapshot object",
			pattern:  "prod/*",
			request:  Request{Resource: "snapshots", Action: "read", Object: "prod/repository/snapshot"},
			expected: false,
		},
		{
			name:     "two level cluster wildcard matches snapshot object",
			pattern:  "prod/*/*",
			request:  Request{Resource: "snapshots", Action: "read", Object: "prod/repository/snapshot"},
			expected: true,
		},
		{
			name:     "two level cluster wildcard does not match one level object",
			pattern:  "prod/*/*",
			request:  Request{Resource: "repositories", Action: "read", Object: "prod/repository"},
			expected: false,
		},
		{
			name:     "cluster prefix wildcard does not match resource object",
			pattern:  "prod*",
			request:  Request{Resource: "indices", Action: "read", Object: "prod/index-users"},
			expected: false,
		},
		{
			name:     "cluster prefix wildcard matches cluster id",
			pattern:  "prod*",
			request:  Request{Resource: "overview", Action: "read", Object: "prod-eu"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authorizer := New(config.RBAC{
				Enabled: true,
				Bindings: []config.RBACBinding{
					{Subject: "alice", Role: "role:test"},
				},
				Policies: []config.RBACPolicy{
					{Subject: "role:test", Resource: "*", Action: "read", Object: tt.pattern, Effect: "allow"},
				},
			})

			assert.Equal(t, tt.expected, authorizer.Allow("alice", nil, tt.request))
		})
	}
}

func FuzzAuthorizerDocumentedObjectPatterns(f *testing.F) {
	f.Add("users", "snapshot")
	f.Add("logs-2026.07.04", "daily")
	f.Add("index", "repo")
	f.Add("a/b", "c/d")
	f.Add("", "")

	f.Fuzz(func(t *testing.T, rawSegment string, rawNested string) {
		segment := rbacFuzzSegment(rawSegment, "users")
		nested := rbacFuzzSegment(rawNested, "snapshot")

		indexPrefixAuthorizer := authorizerForObjectPattern("prod/index-*", "indices", "read")
		assertAllow(t, indexPrefixAuthorizer, Request{Resource: "indices", Action: "read", Object: "prod/index-" + segment})
		assertDeny(t, indexPrefixAuthorizer, Request{Resource: "indices", Action: "read", Object: "prod/app-" + segment})
		assertDeny(t, indexPrefixAuthorizer, Request{Resource: "indices", Action: "read", Object: "prod/index-" + segment + "/settings"})

		oneLevelAuthorizer := authorizerForObjectPattern("prod/*", "indices", "read")
		assertAllow(t, oneLevelAuthorizer, Request{Resource: "indices", Action: "read", Object: "prod/" + segment})
		assertDeny(t, oneLevelAuthorizer, Request{Resource: "indices", Action: "read", Object: "prod/" + segment + "/" + nested})

		twoLevelAuthorizer := authorizerForObjectPattern("prod/*/*", "snapshots", "read")
		assertAllow(t, twoLevelAuthorizer, Request{Resource: "snapshots", Action: "read", Object: "prod/" + segment + "/" + nested})
		assertDeny(t, twoLevelAuthorizer, Request{Resource: "snapshots", Action: "read", Object: "prod/" + segment})

		clusterPrefixAuthorizer := authorizerForObjectPattern("prod*", "overview", "read")
		assertAllow(t, clusterPrefixAuthorizer, Request{Resource: "overview", Action: "read", Object: "prod-" + segment})
		assertDeny(t, clusterPrefixAuthorizer, Request{Resource: "overview", Action: "read", Object: "prod/" + segment})
	})
}

func FuzzAuthorizerIndexPrefixActions(f *testing.F) {
	f.Add("users")
	f.Add("logs-2026.07.04")
	f.Add("app-users")
	f.Add("a/b")
	f.Add("")

	f.Fuzz(func(t *testing.T, rawIndexSuffix string) {
		suffix := rbacFuzzSegment(rawIndexSuffix, "users")
		authorizer := New(config.RBAC{
			Enabled: true,
			Bindings: []config.RBACBinding{
				{Subject: "alice", Role: "role:index-maintainer"},
			},
			Policies: []config.RBACPolicy{
				{Subject: "role:index-maintainer", Resource: "indices", Action: "read", Object: "prod/index-*", Effect: "allow"},
				{Subject: "role:index-maintainer", Resource: "indices", Action: "delete", Object: "prod/index-*", Effect: "allow"},
				{Subject: "role:index-maintainer", Resource: "documents", Action: "write", Object: "prod/index-*", Effect: "allow"},
			},
		})

		matchingIndex := "prod/index-" + suffix
		otherIndex := "prod/app-" + suffix

		assertAllow(t, authorizer, Request{Resource: "indices", Action: "read", Object: matchingIndex})
		assertAllow(t, authorizer, Request{Resource: "indices", Action: "delete", Object: matchingIndex})
		assertAllow(t, authorizer, Request{Resource: "documents", Action: "write", Object: matchingIndex})
		assertDeny(t, authorizer, Request{Resource: "indices", Action: "refresh", Object: matchingIndex})
		assertDeny(t, authorizer, Request{Resource: "indices", Action: "read", Object: otherIndex})
		assertDeny(t, authorizer, Request{Resource: "indices", Action: "delete", Object: otherIndex})
		assertDeny(t, authorizer, Request{Resource: "documents", Action: "write", Object: otherIndex})
	})
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
			name:     "data explorer browse",
			method:   http.MethodGet,
			path:     "/clusters/local-cluster/data_explorer/logs-000001",
			expected: Request{Resource: "documents", Action: "read", Object: "local-cluster/logs-000001"},
		},
		{
			name:     "snapshot restore",
			method:   http.MethodPost,
			path:     "/clusters/local-cluster/snapshots/fs/snap-1/restore",
			expected: Request{Resource: "snapshots", Action: "restore", Object: "local-cluster/fs/snap-1"},
		},
		{
			name:     "overview shard relocation",
			method:   http.MethodPost,
			path:     "/clusters/local-cluster/overview/indices/logs-000001/0/node-1/relocation",
			expected: Request{Resource: "shards", Action: "relocate", Object: "local-cluster/logs-000001"},
		},
		{
			name:     "commons index helper maps to indices",
			method:   http.MethodGet,
			path:     "/clusters/local-cluster/commons/indices/logs-000001/settings",
			expected: Request{Resource: "indices", Action: "read", Object: "local-cluster/logs-000001"},
		},
		{
			name:     "commons index helper without index id stays index scoped",
			method:   http.MethodGet,
			path:     "/clusters/local-cluster/commons/indices",
			expected: Request{Resource: "indices", Action: "read", Object: "local-cluster"},
		},
		{
			name:     "commons nodes helper maps to nodes",
			method:   http.MethodGet,
			path:     "/clusters/local-cluster/commons/nodes/node-1/stats",
			expected: Request{Resource: "nodes", Action: "read", Object: "local-cluster/node-1"},
		},
		{
			name:     "commons nodes helper without node id stays node scoped",
			method:   http.MethodGet,
			path:     "/clusters/local-cluster/commons/nodes",
			expected: Request{Resource: "nodes", Action: "read", Object: "local-cluster"},
		},
		{
			name:     "create index without target falls back to cluster object",
			method:   http.MethodPost,
			path:     "/clusters/local-cluster/create_index",
			expected: Request{Resource: "indices", Action: "create", Object: "local-cluster"},
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

func authorizerForObjectPattern(pattern, resource, action string) *Authorizer {
	return New(config.RBAC{
		Enabled: true,
		Bindings: []config.RBACBinding{
			{Subject: "alice", Role: "role:test"},
		},
		Policies: []config.RBACPolicy{
			{Subject: "role:test", Resource: resource, Action: action, Object: pattern, Effect: "allow"},
		},
	})
}

func assertAllow(t *testing.T, authorizer *Authorizer, request Request) {
	t.Helper()
	if !authorizer.Allow("alice", nil, request) {
		t.Fatalf("expected allow for request %#v", request)
	}
}

func assertDeny(t *testing.T, authorizer *Authorizer, request Request) {
	t.Helper()
	if authorizer.Allow("alice", nil, request) {
		t.Fatalf("expected deny for request %#v", request)
	}
}

func rbacFuzzSegment(raw, fallback string) string {
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-',
			r == '_',
			r == '.':
			b.WriteRune(r)
		}
		if b.Len() >= 48 {
			break
		}
	}
	if b.Len() == 0 {
		return fallback
	}
	return b.String()
}
