package history

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreSaveDeduplicatesAndRedacts(t *testing.T) {
	store := openTestStore(t, 10)
	ctx := context.Background()

	require.NoError(t, store.Save(ctx, "alice", "/_search", "POST", `{"password":"secret"}`))
	require.NoError(t, store.Save(ctx, "alice", "/_search", "POST", `{"password":"secret"}`))

	rows, err := store.All(ctx, "alice")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "/_search", rows[0].Path)
	assert.JSONEq(t, `{"password":"<redacted>"}`, rows[0].Body)
}

func TestStoreLimitsHistoryPerUser(t *testing.T) {
	store := openTestStore(t, 2)
	ctx := context.Background()

	require.NoError(t, store.Save(ctx, "alice", "/first", "GET", "{}"))
	require.NoError(t, store.Save(ctx, "alice", "/second", "GET", "{}"))
	require.NoError(t, store.Save(ctx, "alice", "/third", "GET", "{}"))
	require.NoError(t, store.Save(ctx, "bob", "/bob", "GET", "{}"))

	aliceRows, err := store.All(ctx, "alice")
	require.NoError(t, err)
	require.Len(t, aliceRows, 2)
	assert.Equal(t, "/third", aliceRows[0].Path)
	assert.Equal(t, "/second", aliceRows[1].Path)

	bobRows, err := store.All(ctx, "bob")
	require.NoError(t, err)
	require.Len(t, bobRows, 1)
}

func TestStoreClearOnlyRemovesUserHistory(t *testing.T) {
	store := openTestStore(t, 10)
	ctx := context.Background()

	require.NoError(t, store.Save(ctx, "alice", "/alice", "GET", "{}"))
	require.NoError(t, store.Save(ctx, "bob", "/bob", "GET", "{}"))

	deleted, err := store.Clear(ctx, "alice")
	require.NoError(t, err)
	assert.EqualValues(t, 1, deleted)

	aliceRows, err := store.All(ctx, "alice")
	require.NoError(t, err)
	assert.Empty(t, aliceRows)
	bobRows, err := store.All(ctx, "bob")
	require.NoError(t, err)
	require.Len(t, bobRows, 1)
}

func TestRedactBody_RedactsSensitiveJSONFields(t *testing.T) {
	body := `{
		"password": "secret",
		"nested": {
			"access_token": "token-value",
			"safe": "visible"
		},
		"items": [{"client_secret": "hidden"}]
	}`

	redacted := RedactBody(body)
	var got map[string]any
	assert.NoError(t, json.Unmarshal([]byte(redacted), &got))

	assert.NotContains(t, redacted, "token-value")
	assert.NotContains(t, redacted, "hidden")
	assert.Equal(t, "<redacted>", got["password"])
	nested := got["nested"].(map[string]any)
	assert.Equal(t, "<redacted>", nested["access_token"])
	assert.Equal(t, "visible", nested["safe"])
	items := got["items"].([]any)
	assert.Equal(t, "<redacted>", items[0].(map[string]any)["client_secret"])
}

func TestRedactBody_TruncatesLargeBodies(t *testing.T) {
	body := strings.Repeat("x", maxStoredBodyBytes+1)

	redacted := RedactBody(body)

	assert.Len(t, redacted, maxStoredBodyBytes+len("...<truncated>"))
	assert.True(t, strings.HasSuffix(redacted, "...<truncated>"))
}

func openTestStore(t *testing.T, historySize int) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "history.db"), historySize)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})
	return store
}
