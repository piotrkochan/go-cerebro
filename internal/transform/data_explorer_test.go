package transform

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDataExplorerSearch(t *testing.T) {
	raw := json.RawMessage(`{
		"hits": {
			"total": {"value": 2, "relation": "eq"},
			"hits": [
				{"_id": "1", "_score": 1.5, "_source": {"name": "alpha", "meta": {"status": "ok"}, "empty": null}},
				{"_id": "2", "_score": null, "_source": {"name": "beta", "count": 3}}
			]
		}
	}`)

	out := DataExplorerSearch(raw)

	assert.Equal(t, int64(2), out.Total)
	assert.Equal(t, []string{"_id", "_score", "count", "empty", "meta.status", "name"}, out.Columns)
	assert.JSONEq(t, `"alpha"`, string(out.Rows[0]["name"]))
	assert.JSONEq(t, `"ok"`, string(out.Rows[0]["meta.status"]))
	assert.JSONEq(t, `null`, string(out.Rows[0]["empty"]))
}
