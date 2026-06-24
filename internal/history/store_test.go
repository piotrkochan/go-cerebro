package history

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
