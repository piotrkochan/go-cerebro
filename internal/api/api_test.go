package api

import (
	"testing"

	"github.com/lmenezes/cerebro/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAdHocHostRejectsCredentialsInURL(t *testing.T) {
	err := validateAdHocHost("https://elastic:secret@example.com:9200")

	assert.EqualError(t, err, "credentials in elasticsearch host URL are not allowed")
}

func TestResolveClusterTargetRequiresSlug(t *testing.T) {
	deps := &Deps{Cfg: &config.Config{Hosts: []config.Host{
		{Name: "Local cluster", Host: "http://localhost:9200"},
	}}}

	target, err := deps.resolveClusterTarget(nil, "local-cluster")
	require.NoError(t, err)
	assert.Equal(t, "Local cluster", target.Host.Name)

	_, err = deps.resolveClusterTarget(nil, "Local cluster")
	assert.EqualError(t, err, "unknown elasticsearch cluster slug")
}
