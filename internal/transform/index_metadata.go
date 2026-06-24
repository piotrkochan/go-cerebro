package transform

import "encoding/json"

// Metadata is the payload of the /create_index/get_index_metadata endpoint.
type Metadata struct {
	Mappings json.RawMessage `json:"mappings" doc:"Index mappings (null when not found)."`
	Settings json.RawMessage `json:"settings" doc:"Index settings (null when not found)."`
}

// IndexMetadata picks first "mappings" and "settings" found anywhere in the cluster state metadata.
// Port of app/models/IndexMetadata.scala.
func IndexMetadata(raw json.RawMessage) Metadata {
	return Metadata{
		Mappings: findFirst(raw, "mappings"),
		Settings: findFirst(raw, "settings"),
	}
}

// findFirst recursively searches JSON for the first occurrence of a top-level key.
func findFirst(raw json.RawMessage, key string) json.RawMessage {
	var node any
	if err := json.Unmarshal(raw, &node); err != nil {
		return json.RawMessage("null")
	}
	res := walkFirst(node, key)
	if res == nil {
		return json.RawMessage("null")
	}
	b, _ := json.Marshal(res)
	return b
}

func walkFirst(n any, key string) any {
	switch v := n.(type) {
	case map[string]any:
		if found, ok := v[key]; ok {
			return found
		}
		for _, child := range v {
			if r := walkFirst(child, key); r != nil {
				return r
			}
		}
	case []any:
		for _, child := range v {
			if r := walkFirst(child, key); r != nil {
				return r
			}
		}
	}
	return nil
}
