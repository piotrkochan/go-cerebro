package transform

import "encoding/json"

// OpenIndices returns "index" values for entries with status=="open". Port of models/analysis/OpenIndices.scala.
func OpenIndices(raw json.RawMessage) []string {
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		return []string{}
	}
	out := []string{}
	for _, r := range rows {
		if status, _ := r["status"].(string); status == "open" {
			if idx, ok := r["index"].(string); ok {
				out = append(out, idx)
			}
		}
	}
	return out
}

// IndexAnalyzers returns analyzer names defined for an index. Port of models/analysis/IndexAnalyzers.scala.
func IndexAnalyzers(index string, raw json.RawMessage) []string {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return []string{}
	}
	indexBlock, ok := root[index]
	if !ok {
		return []string{}
	}
	var settings struct {
		Settings struct {
			Index struct {
				Analysis struct {
					Analyzer map[string]json.RawMessage `json:"analyzer"`
				} `json:"analysis"`
			} `json:"index"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(indexBlock, &settings); err != nil {
		return []string{}
	}
	names := make([]string, 0, len(settings.Settings.Index.Analysis.Analyzer))
	for n := range settings.Settings.Index.Analysis.Analyzer {
		names = append(names, n)
	}
	return names
}

// Tokens extracts the "tokens" field from an _analyze response. The token objects are kept
// verbatim (their fields vary across Elasticsearch versions). Port of models/analysis/Tokens.scala.
func Tokens(raw json.RawMessage) json.RawMessage {
	var obj struct {
		Tokens json.RawMessage `json:"tokens"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil || len(obj.Tokens) == 0 {
		return json.RawMessage(`[]`)
	}
	return obj.Tokens
}

var analyzableFieldTypes = map[string]bool{"string": true, "text": true}

// IndexFields returns analyzable text fields for an index, traversing properties recursively.
// Port of models/analysis/IndexFields.scala.
func IndexFields(index string, raw json.RawMessage) []string {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return []string{}
	}
	indexBlock, ok := root[index]
	if !ok {
		return []string{}
	}
	var withMappings struct {
		Mappings json.RawMessage `json:"mappings"`
	}
	if err := json.Unmarshal(indexBlock, &withMappings); err != nil {
		return []string{}
	}
	var mappings map[string]json.RawMessage
	if err := json.Unmarshal(withMappings.Mappings, &mappings); err != nil {
		return []string{}
	}
	out := []string{}
	if len(mappings) == 1 {
		if props, ok := mappings["properties"]; ok {
			return append(out, extractProperties(props, "")...)
		}
	}
	// ES < 7 — multiple doc types
	for _, docType := range mappings {
		var dt struct {
			Properties json.RawMessage `json:"properties"`
		}
		if err := json.Unmarshal(docType, &dt); err == nil && len(dt.Properties) > 0 {
			out = append(out, extractProperties(dt.Properties, "")...)
		}
	}
	return out
}

func extractProperties(raw json.RawMessage, prefix string) []string {
	var props map[string]json.RawMessage
	if err := json.Unmarshal(raw, &props); err != nil {
		return nil
	}
	out := []string{}
	for name, val := range props {
		var node struct {
			Type       string                     `json:"type"`
			Properties json.RawMessage            `json:"properties"`
			Fields     map[string]json.RawMessage `json:"fields"`
		}
		_ = json.Unmarshal(val, &node)
		full := name
		if prefix != "" {
			full = prefix + "." + name
		}
		if len(node.Properties) > 0 {
			out = append(out, extractProperties(node.Properties, full)...)
			continue
		}
		if len(node.Fields) > 0 {
			added := []string{}
			for fname, fraw := range node.Fields {
				var ftype struct {
					Type string `json:"type"`
				}
				_ = json.Unmarshal(fraw, &ftype)
				if analyzableFieldTypes[ftype.Type] {
					added = append(added, full+"."+fname)
				}
			}
			if analyzableFieldTypes[node.Type] {
				added = append(added, full)
			}
			out = append(out, added...)
			continue
		}
		if analyzableFieldTypes[node.Type] {
			out = append(out, full)
		}
	}
	return out
}
