package transform

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestILMPoliciesExtractsPhasesAndUsage(t *testing.T) {
	raw := json.RawMessage(`{
		"logs-policy": {
			"version": 3,
			"modified_date": "2026-07-01T10:00:00Z",
			"policy": {
				"phases": {
					"delete": {"min_age": "30d"},
					"hot": {"actions": {"rollover": {"max_age": "1d"}}},
					"warm": {"min_age": "7d"},
					"custom": {}
				}
			},
			"in_use_by": {
				"indices": ["logs-000001"],
				"data_streams": ["logs"],
				"composable_templates": ["logs-template"]
			}
		},
		"empty-policy": {
			"policy": {"phases": {}}
		}
	}`)

	got := ilmPoliciesByName(ILMPolicies(raw))

	require.Contains(t, got, "logs-policy")
	policy := got["logs-policy"]
	assert.Equal(t, "logs-policy", policy.Name)
	assert.EqualValues(t, 3, policy.Version)
	assert.Equal(t, []string{"hot", "warm", "delete", "custom"}, policy.Phases)
	assert.Equal(t, []string{"logs-000001"}, policy.InUseBy.Indices)
	assert.Equal(t, []string{"logs"}, policy.InUseBy.DataStreams)
	assert.Equal(t, []string{"logs-template"}, policy.InUseBy.ComposableTemplates)
	assert.JSONEq(t, `{"phases":{"delete":{"min_age":"30d"},"hot":{"actions":{"rollover":{"max_age":"1d"}}},"warm":{"min_age":"7d"},"custom":{}}}`, string(policy.Policy))

	require.Contains(t, got, "empty-policy")
	assert.Empty(t, got["empty-policy"].Phases)
	assert.NotNil(t, got["empty-policy"].InUseBy.Indices)
	assert.NotNil(t, got["empty-policy"].InUseBy.DataStreams)
	assert.NotNil(t, got["empty-policy"].InUseBy.ComposableTemplates)
}

func TestILMPoliciesReturnsEmptyOnInvalidJSON(t *testing.T) {
	assert.Empty(t, ILMPolicies(json.RawMessage(`{`)))
}

func ilmPoliciesByName(items []ILMPolicy) map[string]ILMPolicy {
	out := make(map[string]ILMPolicy, len(items))
	for _, item := range items {
		out[item.Name] = item
	}
	return out
}
