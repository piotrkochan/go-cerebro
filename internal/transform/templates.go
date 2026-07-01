package transform

import "encoding/json"

// TemplateSummary is a lightweight template row for list pages.
type TemplateSummary struct {
	DataStream bool   `json:"data_stream,omitempty" doc:"Whether the template creates data streams instead of regular indices."`
	Kind       string `json:"kind" doc:"Template kind: index, component, or legacy."`
	Managed    bool   `json:"managed" doc:"Whether the template appears to be managed by Elasticsearch or an integration."`
	Name       string `json:"name" doc:"Template name."`
	Pattern    string `json:"pattern,omitempty" doc:"Template index pattern, when applicable."`
}

// Template is one index template entry.
type Template struct {
	Kind     string          `json:"kind" doc:"Template kind: index, component, or legacy."`
	Name     string          `json:"name" doc:"Template name."`
	Template json.RawMessage `json:"template" doc:"Raw template definition."`
}

// TemplateSummaries returns a lightweight legacy _template list.
func TemplateSummaries(raw json.RawMessage) []TemplateSummary {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return []TemplateSummary{}
	}
	out := make([]TemplateSummary, 0, len(m))
	for name, tmpl := range m {
		out = append(out, TemplateSummary{
			Kind:    "legacy",
			Managed: managedTemplate(tmpl, name),
			Name:    name,
			Pattern: legacyTemplatePattern(tmpl),
		})
	}
	return out
}

// Templates returns the template list from a raw legacy _template response. Port of models/templates/Templates.scala.
func Templates(raw json.RawMessage) []Template {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return []Template{}
	}
	out := make([]Template, 0, len(m))
	for name, tmpl := range m {
		out = append(out, Template{Kind: "legacy", Name: name, Template: tmpl})
	}
	return out
}

// LegacyTemplate returns one legacy template by name from _template/{name}.
func LegacyTemplate(raw json.RawMessage, name string) (Template, bool) {
	templates := Templates(raw)
	for _, template := range templates {
		if template.Name == name {
			return template, true
		}
	}
	return Template{}, false
}

// ComposableIndexTemplateSummaries returns lightweight templates from _index_template.
func ComposableIndexTemplateSummaries(raw json.RawMessage) []TemplateSummary {
	var root struct {
		IndexTemplates []struct {
			Name          string          `json:"name"`
			IndexTemplate json.RawMessage `json:"index_template"`
		} `json:"index_templates"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return []TemplateSummary{}
	}
	out := make([]TemplateSummary, 0, len(root.IndexTemplates))
	for _, item := range root.IndexTemplates {
		out = append(out, TemplateSummary{
			DataStream: composableTemplateCreatesDataStream(item.IndexTemplate),
			Kind:       "index",
			Managed:    managedTemplate(item.IndexTemplate, item.Name),
			Name:       item.Name,
			Pattern:    composableTemplatePattern(item.IndexTemplate),
		})
	}
	return out
}

// ComposableIndexTemplates returns templates from _index_template.
func ComposableIndexTemplates(raw json.RawMessage) []Template {
	var root struct {
		IndexTemplates []struct {
			Name          string          `json:"name"`
			IndexTemplate json.RawMessage `json:"index_template"`
		} `json:"index_templates"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return []Template{}
	}
	out := make([]Template, 0, len(root.IndexTemplates))
	for _, item := range root.IndexTemplates {
		out = append(out, Template{Kind: "index", Name: item.Name, Template: item.IndexTemplate})
	}
	return out
}

// ComposableIndexTemplate returns one template from _index_template/{name}.
func ComposableIndexTemplate(raw json.RawMessage, name string) (Template, bool) {
	templates := ComposableIndexTemplates(raw)
	for _, template := range templates {
		if template.Name == name {
			return template, true
		}
	}
	return Template{}, false
}

// ComponentTemplateSummaries returns lightweight component templates from _component_template.
func ComponentTemplateSummaries(raw json.RawMessage) []TemplateSummary {
	var root struct {
		ComponentTemplates []struct {
			Name              string          `json:"name"`
			ComponentTemplate json.RawMessage `json:"component_template"`
		} `json:"component_templates"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return []TemplateSummary{}
	}
	out := make([]TemplateSummary, 0, len(root.ComponentTemplates))
	for _, item := range root.ComponentTemplates {
		out = append(out, TemplateSummary{
			Kind:    "component",
			Managed: managedTemplate(item.ComponentTemplate, item.Name),
			Name:    item.Name,
		})
	}
	return out
}

// ComponentTemplates returns templates from _component_template.
func ComponentTemplates(raw json.RawMessage) []Template {
	var root struct {
		ComponentTemplates []struct {
			Name              string          `json:"name"`
			ComponentTemplate json.RawMessage `json:"component_template"`
		} `json:"component_templates"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return []Template{}
	}
	out := make([]Template, 0, len(root.ComponentTemplates))
	for _, item := range root.ComponentTemplates {
		out = append(out, Template{Kind: "component", Name: item.Name, Template: item.ComponentTemplate})
	}
	return out
}

// ComponentTemplate returns one component template from _component_template/{name}.
func ComponentTemplate(raw json.RawMessage, name string) (Template, bool) {
	templates := ComponentTemplates(raw)
	for _, template := range templates {
		if template.Name == name {
			return template, true
		}
	}
	return Template{}, false
}

func legacyTemplatePattern(raw json.RawMessage) string {
	var body struct {
		Template string `json:"template"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return ""
	}
	return body.Template
}

func composableTemplatePattern(raw json.RawMessage) string {
	var body struct {
		IndexPatterns []string `json:"index_patterns"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return ""
	}
	if len(body.IndexPatterns) == 0 {
		return ""
	}
	out := body.IndexPatterns[0]
	for _, pattern := range body.IndexPatterns[1:] {
		out += ", " + pattern
	}
	return out
}

func composableTemplateCreatesDataStream(raw json.RawMessage) bool {
	var body struct {
		DataStream json.RawMessage `json:"data_stream"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return false
	}
	return len(body.DataStream) > 0 && string(body.DataStream) != "null"
}

func managedTemplate(raw json.RawMessage, name string) bool {
	if len(name) > 0 && name[0] == '.' {
		return true
	}
	var body struct {
		Managed bool `json:"managed"`
		Meta    struct {
			Managed   any    `json:"managed"`
			ManagedBy string `json:"managed_by"`
			System    any    `json:"system"`
		} `json:"_meta"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return false
	}
	if body.Managed || body.Meta.ManagedBy != "" {
		return true
	}
	return truthy(body.Meta.Managed) || truthy(body.Meta.System)
}

func truthy(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v == "true"
	default:
		return false
	}
}
