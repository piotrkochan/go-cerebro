package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeDataExplorerQuery(t *testing.T) {
	assert.Equal(t, "items_count:7", normalizeDataExplorerQuery("items_count: 7"))
	assert.Equal(t, `status:"paid" AND items_count:7`, normalizeDataExplorerQuery(`status: "paid" AND items_count: 7`))
	assert.Equal(t, `items_count:>=7 AND status:paid`, normalizeKQLQuery(`items_count >= 7 and status: paid`))
}

func TestDataExplorerQueryNormalizesFieldSpacing(t *testing.T) {
	in := &DataExplorerSearchIn{}
	in.Body.Query = "items_count: 7"

	raw, err := dataExplorerQuery(in)
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, json.Unmarshal(raw, &body))
	query := body["query"].(map[string]any)["query_string"].(map[string]any)
	assert.Equal(t, "items_count:7", query["query"])
}

func TestDataExplorerIndexReadOnly(t *testing.T) {
	raw := json.RawMessage(`{
		"orders": {
			"settings": {
				"index": {
					"blocks": {
						"write": "true"
					}
				}
			}
		}
	}`)

	assert.True(t, dataExplorerIndexReadOnly(raw, "orders"))
}

func TestDataExplorerIndexReadOnlyFlatSettings(t *testing.T) {
	raw := json.RawMessage(`{
		"orders": {
			"settings": {
				"index.blocks.read_only_allow_delete": "true"
			}
		}
	}`)

	assert.True(t, dataExplorerIndexReadOnly(raw, "orders"))
}
