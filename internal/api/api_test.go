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

func TestResolveClusterTargetAllowsAdHocURLWhenEnabled(t *testing.T) {
	deps := &Deps{Cfg: &config.Config{ES: config.ES{AllowAdHocHosts: true}}}

	target, err := deps.resolveClusterTarget(nil, "http://10.0.2.4:9200")

	require.NoError(t, err)
	assert.Equal(t, "http://10.0.2.4:9200", target.Host.Name)
	assert.Equal(t, "http://10.0.2.4:9200", target.Host.Host)
}

func TestResolveClusterTargetRejectsAdHocURLWhenDisabled(t *testing.T) {
	deps := &Deps{Cfg: &config.Config{}}

	_, err := deps.resolveClusterTarget(nil, "http://10.0.2.4:9200")

	assert.EqualError(t, err, "unknown elasticsearch cluster slug")
}

func TestResolveClusterTargetRejectsInvalidAdHocURLAsUnknownSlug(t *testing.T) {
	deps := &Deps{Cfg: &config.Config{ES: config.ES{AllowAdHocHosts: true}}}

	_, err := deps.resolveClusterTarget(nil, "local-cluster")

	assert.EqualError(t, err, "unknown elasticsearch cluster slug")
}
