package transform

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAliases_Flattens(t *testing.T) {
	in := json.RawMessage(`{
		"books": {"aliases": {"latest": {"filter": {"term": {"y": 2024}}}}},
		"empty": {"aliases": {}}
	}`)
	out := Aliases(in)
	require.Len(t, out, 1)
	assert.Equal(t, "latest", out[0].Alias)
	assert.Equal(t, "books", out[0].Index)
	assert.JSONEq(t, `{"term": {"y": 2024}}`, string(out[0].Filter))
}

func TestAliases_NullForAbsentProps(t *testing.T) {
	in := json.RawMessage(`{"books": {"aliases": {"latest": {}}}}`)
	b, err := json.Marshal(Aliases(in))
	require.NoError(t, err)
	assert.JSONEq(t, `[{"alias":"latest","index":"books","filter":null,"search_routing":null,"index_routing":null}]`, string(b))
}

func TestAutocompletionIndices_Distinct(t *testing.T) {
	in := json.RawMessage(`{
		"a": {"aliases": {"a-alias": {}}},
		"b": {"aliases": {}}
	}`)
	assert.ElementsMatch(t, []string{"a", "a-alias", "b"}, AutocompletionIndices(in))
}
