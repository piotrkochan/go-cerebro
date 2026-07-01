package transform

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpenIndices(t *testing.T) {
	raw := json.RawMessage(`[
		{"status": "open", "index": "logs"},
		{"status": "close", "index": "archive"},
		{"status": "open", "index": "users"}
	]`)

	assert.Equal(t, []string{"logs", "users"}, OpenIndices(raw))
	assert.Empty(t, OpenIndices(json.RawMessage(`{`)))
}

func TestIndexAnalyzers(t *testing.T) {
	raw := json.RawMessage(`{
		"logs": {
			"settings": {
				"index": {
					"analysis": {
						"analyzer": {
							"custom": {},
							"edge": {}
						}
					}
				}
			}
		}
	}`)

	assert.ElementsMatch(t, []string{"custom", "edge"}, IndexAnalyzers("logs", raw))
	assert.Empty(t, IndexAnalyzers("missing", raw))
}

func TestTokens(t *testing.T) {
	assert.JSONEq(t, `[{"token":"hello"}]`, string(Tokens(json.RawMessage(`{"tokens":[{"token":"hello"}]}`))))
	assert.JSONEq(t, `[]`, string(Tokens(json.RawMessage(`{"detail":{}}`))))
	assert.JSONEq(t, `[]`, string(Tokens(json.RawMessage(`{`))))
}

func TestIndexFieldsTraversesModernAndLegacyMappings(t *testing.T) {
	modern := json.RawMessage(`{
		"logs": {
			"mappings": {
				"properties": {
					"message": {"type": "text"},
					"host": {"type": "keyword"},
					"user": {
						"properties": {
							"name": {"type": "text"},
							"id": {"type": "keyword"}
						}
					},
					"title": {
						"type": "text",
						"fields": {
							"raw": {"type": "keyword"},
							"stemmed": {"type": "text"}
						}
					}
				}
			}
		}
	}`)
	legacy := json.RawMessage(`{
		"logs": {
			"mappings": {
				"_doc": {
					"properties": {
						"body": {"type": "string"},
						"code": {"type": "keyword"}
					}
				}
			}
		}
	}`)

	assert.ElementsMatch(t, []string{"message", "user.name", "title.stemmed", "title"}, IndexFields("logs", modern))
	assert.ElementsMatch(t, []string{"body"}, IndexFields("logs", legacy))
	assert.Empty(t, IndexFields("missing", modern))
}
