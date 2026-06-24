package transform

import "encoding/json"

// CommonsIndices returns "index" values from /_cat/indices. Port of models/commons/Indices.scala.
func CommonsIndices(raw json.RawMessage) []string {
	return collectByKey(raw, "index")
}

// CommonsNodes returns "name" values from /_cat/nodes. Port of models/commons/Nodes.scala.
func CommonsNodes(raw json.RawMessage) []string {
	return collectByKey(raw, "name")
}

func collectByKey(raw json.RawMessage, key string) []string {
	var node any
	if err := json.Unmarshal(raw, &node); err != nil {
		return []string{}
	}
	out := []string{}
	walkAll(node, key, &out)
	return out
}

func walkAll(n any, key string, out *[]string) {
	switch v := n.(type) {
	case map[string]any:
		for k, child := range v {
			if k == key {
				*out = append(*out, textValue(child))
			}
			walkAll(child, key, out)
		}
	case []any:
		for _, child := range v {
			walkAll(child, key, out)
		}
	}
}

func textValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
