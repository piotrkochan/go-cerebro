package transform

import "encoding/json"

// Template is one index template entry.
type Template struct {
	Name     string          `json:"name" doc:"Template name."`
	Template json.RawMessage `json:"template" doc:"Raw template definition."`
}

// Templates returns the template list from a raw _template response. Port of models/templates/Templates.scala.
func Templates(raw json.RawMessage) []Template {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return []Template{}
	}
	out := make([]Template, 0, len(m))
	for name, tmpl := range m {
		out = append(out, Template{Name: name, Template: tmpl})
	}
	return out
}
