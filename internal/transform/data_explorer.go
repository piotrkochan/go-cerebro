package transform

import (
	"encoding/json"
	"sort"
	"strings"
)

type DataExplorerResult struct {
	Columns []string                     `json:"columns" doc:"Columns discovered from the current result page."`
	Rows    []map[string]json.RawMessage `json:"rows" doc:"Flattened documents for tabular display."`
	Total   int64                        `json:"total" doc:"Total matching documents reported by Elasticsearch."`
}

func DataExplorerSearch(raw json.RawMessage) DataExplorerResult {
	var payload struct {
		Hits struct {
			Total any `json:"total"`
			Hits  []struct {
				ID     string                     `json:"_id"`
				Score  *float64                   `json:"_score"`
				Source map[string]json.RawMessage `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return DataExplorerResult{Columns: []string{}, Rows: []map[string]json.RawMessage{}}
	}

	columns := map[string]struct{}{"_id": {}, "_score": {}}
	rows := make([]map[string]json.RawMessage, 0, len(payload.Hits.Hits))
	for _, hit := range payload.Hits.Hits {
		row := map[string]json.RawMessage{}
		row["_id"], _ = json.Marshal(hit.ID)
		row["_source"], _ = json.Marshal(hit.Source)
		if hit.Score != nil {
			row["_score"], _ = json.Marshal(*hit.Score)
		} else {
			row["_score"] = json.RawMessage("null")
		}
		for key, value := range flattenSource("", hit.Source) {
			row[key] = value
			columns[key] = struct{}{}
		}
		rows = append(rows, row)
	}

	return DataExplorerResult{
		Columns: sortedColumns(columns),
		Rows:    rows,
		Total:   totalHits(payload.Hits.Total),
	}
}

func flattenSource(prefix string, source map[string]json.RawMessage) map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	for key, value := range source {
		name := key
		if prefix != "" {
			name = prefix + "." + key
		}
		var nested map[string]json.RawMessage
		if err := json.Unmarshal(value, &nested); err == nil && nested != nil {
			for nestedKey, nestedValue := range flattenSource(name, nested) {
				out[nestedKey] = nestedValue
			}
			continue
		}
		out[name] = value
	}
	return out
}

func sortedColumns(columns map[string]struct{}) []string {
	out := make([]string, 0, len(columns))
	for column := range columns {
		out = append(out, column)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i] == "_id" {
			return true
		}
		if out[j] == "_id" {
			return false
		}
		if out[i] == "_score" {
			return true
		}
		if out[j] == "_score" {
			return false
		}
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out
}

func totalHits(total any) int64 {
	switch v := total.(type) {
	case float64:
		return int64(v)
	case map[string]any:
		if n, ok := v["value"].(float64); ok {
			return int64(n)
		}
	}
	return 0
}
