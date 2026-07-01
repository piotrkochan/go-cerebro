package transform

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositories(t *testing.T) {
	raw := json.RawMessage(`{
		"fs-repo": {"type": "fs", "settings": {"location": "/snapshots", "compress": true}},
		"s3-repo": {"type": "s3"}
	}`)

	got, err := Repositories(raw)
	require.NoError(t, err)
	repos := repositoriesByName(got)

	require.Contains(t, repos, "fs-repo")
	assert.Equal(t, "fs", repos["fs-repo"].Type)
	assert.JSONEq(t, `{"location":"/snapshots","compress":true}`, string(repos["fs-repo"].Settings))

	require.Contains(t, repos, "s3-repo")
	assert.Equal(t, "s3", repos["s3-repo"].Type)
	assert.JSONEq(t, `{}`, string(repos["s3-repo"].Settings))
}

func TestRepositoriesReturnsJSONError(t *testing.T) {
	got, err := Repositories(json.RawMessage(`{`))
	require.Error(t, err)
	assert.Nil(t, got)
}

func repositoriesByName(items []Repository) map[string]Repository {
	out := make(map[string]Repository, len(items))
	for _, item := range items {
		out[item.Name] = item
	}
	return out
}
