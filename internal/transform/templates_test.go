package transform

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateSummaries(t *testing.T) {
	raw := json.RawMessage(`{
		"app-template": {
			"template": "app-*",
			"settings": {"number_of_shards": 1}
		},
		".system-template": {
			"template": ".system-*"
		},
		"managed-template": {
			"template": "managed-*",
			"_meta": {"managed": "true"}
		}
	}`)

	got := templateSummariesByName(TemplateSummaries(raw))

	require.Contains(t, got, "app-template")
	assert.Equal(t, TemplateSummary{Kind: "legacy", Name: "app-template", Pattern: "app-*"}, got["app-template"])

	require.Contains(t, got, ".system-template")
	assert.True(t, got[".system-template"].Managed)
	assert.Equal(t, ".system-*", got[".system-template"].Pattern)

	require.Contains(t, got, "managed-template")
	assert.True(t, got["managed-template"].Managed)
}

func TestComposableIndexTemplateSummaries(t *testing.T) {
	raw := json.RawMessage(`{
		"index_templates": [
			{
				"name": "logs-template",
				"index_template": {
					"index_patterns": ["logs-*", "events-*"],
					"data_stream": {},
					"_meta": {"managed_by": "fleet"}
				}
			},
			{
				"name": "plain-template",
				"index_template": {
					"index_patterns": ["plain-*"]
				}
			}
		]
	}`)

	got := ComposableIndexTemplateSummaries(raw)

	require.Len(t, got, 2)
	assert.Equal(t, TemplateSummary{
		DataStream: true,
		Kind:       "index",
		Managed:    true,
		Name:       "logs-template",
		Pattern:    "logs-*, events-*",
	}, got[0])
	assert.Equal(t, TemplateSummary{
		Kind:    "index",
		Name:    "plain-template",
		Pattern: "plain-*",
	}, got[1])
}

func TestComponentTemplates(t *testing.T) {
	raw := json.RawMessage(`{
		"component_templates": [
			{"name": "settings", "component_template": {"template": {"settings": {"number_of_shards": 1}}}},
			{"name": ".managed", "component_template": {"template": {"mappings": {}}}}
		]
	}`)

	summaries := ComponentTemplateSummaries(raw)
	require.Len(t, summaries, 2)
	assert.Equal(t, TemplateSummary{Kind: "component", Name: "settings"}, summaries[0])
	assert.Equal(t, TemplateSummary{Kind: "component", Managed: true, Name: ".managed"}, summaries[1])

	template, ok := ComponentTemplate(raw, "settings")
	require.True(t, ok)
	assert.Equal(t, "component", template.Kind)
	assert.Equal(t, "settings", template.Name)
	assert.JSONEq(t, `{"template":{"settings":{"number_of_shards":1}}}`, string(template.Template))
}

func TestTemplateLookupsReturnFalseForMissingTemplate(t *testing.T) {
	legacyRaw := json.RawMessage(`{"app": {"template": "app-*"}}`)
	composableRaw := json.RawMessage(`{"index_templates": [{"name": "app", "index_template": {"index_patterns": ["app-*"]}}]}`)
	componentRaw := json.RawMessage(`{"component_templates": [{"name": "settings", "component_template": {"template": {}}}]}`)

	_, ok := LegacyTemplate(legacyRaw, "missing")
	assert.False(t, ok)

	_, ok = ComposableIndexTemplate(composableRaw, "missing")
	assert.False(t, ok)

	_, ok = ComponentTemplate(componentRaw, "missing")
	assert.False(t, ok)
}

func TestTemplateTransformsReturnEmptyOnInvalidJSON(t *testing.T) {
	invalid := json.RawMessage(`{`)

	assert.Empty(t, TemplateSummaries(invalid))
	assert.Empty(t, Templates(invalid))
	assert.Empty(t, ComposableIndexTemplateSummaries(invalid))
	assert.Empty(t, ComposableIndexTemplates(invalid))
	assert.Empty(t, ComponentTemplateSummaries(invalid))
	assert.Empty(t, ComponentTemplates(invalid))
}

func templateSummariesByName(items []TemplateSummary) map[string]TemplateSummary {
	out := make(map[string]TemplateSummary, len(items))
	for _, item := range items {
		out[item.Name] = item
	}
	return out
}
