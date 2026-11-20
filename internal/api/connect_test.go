package api

import (
	"context"
	"testing"

	"github.com/lmenezes/cerebro/internal/auth"
	"github.com/lmenezes/cerebro/internal/config"
	"github.com/lmenezes/cerebro/internal/rbac"
	"github.com/stretchr/testify/assert"
)

func connectTestDeps(hide bool) *Deps {
	rbacCfg := config.RBAC{
		Enabled:            true,
		HideForbiddenHosts: hide,
		Bindings: []config.RBACBinding{
			{Subject: "group:team-a", Role: "role:a-reader"},
			{Subject: "admin", Role: "role:admin"},
		},
		Policies: []config.RBACPolicy{
			{Subject: "role:admin", Resource: "*", Action: "*", Object: "*", Effect: "allow"},
			{Subject: "role:a-reader", Resource: "*", Action: "read", Object: "a", Effect: "allow"},
			{Subject: "role:a-reader", Resource: "*", Action: "read", Object: "a/*", Effect: "allow"},
		},
	}
	return &Deps{
		Cfg: &config.Config{
			Hosts: []config.Host{
				{ID: "a", Name: "Cluster A", Host: "http://a:9200"},
				{ID: "b", Name: "Cluster B", Host: "http://b:9200"},
			},
			RBAC: rbacCfg,
		},
		RBAC: rbac.New(rbacCfg),
	}
}

func slugs(refs []config.HostRef) []string {
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		out = append(out, r.Slug)
	}
	return out
}

func TestVisibleHostRefsFiltersByRBACWhenEnabled(t *testing.T) {
	deps := connectTestDeps(true)

	ctx := auth.WithIdentity(context.Background(), auth.Identity{Username: "alice", Groups: []string{"team-a"}})
	assert.Equal(t, []string{"a"}, slugs(deps.visibleHostRefs(ctx)))

	ctx = auth.WithIdentity(context.Background(), auth.Identity{Username: "admin"})
	assert.Equal(t, []string{"a", "b"}, slugs(deps.visibleHostRefs(ctx)))

	ctx = auth.WithIdentity(context.Background(), auth.Identity{Username: "nobody"})
	assert.Empty(t, deps.visibleHostRefs(ctx))
}

func TestVisibleHostRefsListsAllHostsByDefault(t *testing.T) {
	deps := connectTestDeps(false)

	ctx := auth.WithIdentity(context.Background(), auth.Identity{Username: "nobody"})
	assert.Equal(t, []string{"a", "b"}, slugs(deps.visibleHostRefs(ctx)))
}

func TestVisibleHostRefsListsAllHostsWhenRBACDisabled(t *testing.T) {
	deps := connectTestDeps(true)
	deps.Cfg.RBAC.Enabled = false
	deps.RBAC = rbac.New(deps.Cfg.RBAC)

	assert.Equal(t, []string{"a", "b"}, slugs(deps.visibleHostRefs(context.Background())))
}
