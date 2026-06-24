package transform

import (
	"encoding/json"
)

// Alias is one flattened alias entry of an index.
type Alias struct {
	Alias         string          `json:"alias" doc:"Alias name."`
	Index         string          `json:"index" doc:"Index the alias points at."`
	Filter        json.RawMessage `json:"filter" doc:"Optional alias filter query (null when absent)."`
	SearchRouting json.RawMessage `json:"search_routing" doc:"Optional search routing value (null when absent)."`
	IndexRouting  json.RawMessage `json:"index_routing" doc:"Optional index routing value (null when absent)."`
}

// Aliases flattens raw _aliases response into a flat array of alias entries.
// Port of app/models/Aliases.scala.
func Aliases(raw json.RawMessage) []Alias {
	var indices map[string]struct {
		Aliases map[string]json.RawMessage `json:"aliases"`
	}
	if err := json.Unmarshal(raw, &indices); err != nil {
		return []Alias{}
	}
	out := []Alias{}
	for index, info := range indices {
		if len(info.Aliases) == 0 {
			continue
		}
		for alias, props := range info.Aliases {
			var p map[string]json.RawMessage
			_ = json.Unmarshal(props, &p)
			out = append(out, Alias{
				Alias:         alias,
				Index:         index,
				Filter:        p["filter"],
				SearchRouting: p["search_routing"],
				IndexRouting:  p["index_routing"],
			})
		}
	}
	return out
}

// AutocompletionIndices returns distinct list of index names + alias names.
// Port of app/models/rest/AutocompletionIndices.scala.
func AutocompletionIndices(raw json.RawMessage) []string {
	var indices map[string]struct {
		Aliases map[string]json.RawMessage `json:"aliases"`
	}
	if err := json.Unmarshal(raw, &indices); err != nil {
		return []string{}
	}
	seen := map[string]bool{}
	out := []string{}
	for idx, info := range indices {
		if !seen[idx] {
			seen[idx] = true
			out = append(out, idx)
		}
		for alias := range info.Aliases {
			if !seen[alias] {
				seen[alias] = true
				out = append(out, alias)
			}
		}
	}
	return out
}
