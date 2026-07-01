package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeTemplateKind(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "index", in: "index", want: "index"},
		{name: "component with whitespace", in: " Component ", want: "component"},
		{name: "legacy uppercase", in: "LEGACY", want: "legacy"},
		{name: "unknown", in: "data-stream", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeTemplateKind(tt.in))
		})
	}
}

func TestTemplateDetailDispatchesByKind(t *testing.T) {
	tests := []struct {
		name string
		kind string
		raw  json.RawMessage
		want string
	}{
		{
			name: "component",
			kind: "component",
			raw:  json.RawMessage(`{"component_templates":[{"name":"settings","component_template":{"template":{"settings":{"number_of_shards":1}}}}]}`),
			want: `{"template":{"settings":{"number_of_shards":1}}}`,
		},
		{
			name: "legacy",
			kind: "legacy",
			raw:  json.RawMessage(`{"settings":{"template":"settings-*","settings":{"number_of_shards":1}}}`),
			want: `{"template":"settings-*","settings":{"number_of_shards":1}}`,
		},
		{
			name: "index default",
			kind: "",
			raw:  json.RawMessage(`{"index_templates":[{"name":"settings","index_template":{"index_patterns":["settings-*"]}}]}`),
			want: `{"index_patterns":["settings-*"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, ok := templateDetail(tt.kind, tt.raw, "settings")
			require.True(t, ok)
			assert.JSONEq(t, tt.want, string(template.Template))
		})
	}
}

func TestTemplateDetailReturnsFalseWhenMissing(t *testing.T) {
	template, ok := templateDetail("component", json.RawMessage(`{"component_templates":[]}`), "missing")

	assert.False(t, ok)
	assert.Empty(t, template.Name)
}
