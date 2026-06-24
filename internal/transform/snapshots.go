package transform

import "encoding/json"

// SnapshotsList extracts the "snapshots" array from a snapshot response. The snapshot
// objects are kept verbatim (their fields vary across Elasticsearch versions).
// Port of models/snapshot/Snapshots.scala.
func SnapshotsList(raw json.RawMessage) json.RawMessage {
	var obj struct {
		Snapshots json.RawMessage `json:"snapshots"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil || len(obj.Snapshots) == 0 {
		return json.RawMessage(`[]`)
	}
	return obj.Snapshots
}

// SnapshotRepositories returns the repository names. Port of models/snapshot/Repositories.scala.
func SnapshotRepositories(raw json.RawMessage) []string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return []string{}
	}
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	return names
}

// SnapshotIndex is one index entry available for snapshotting.
type SnapshotIndex struct {
	Name    string `json:"name" doc:"Index name."`
	Special bool   `json:"special" doc:"Whether the index is dot-prefixed (system index)."`
}

// SnapshotIndices transforms _cat/indices response into a list of snapshot index entries.
// Port of models/snapshot/Indices.scala.
func SnapshotIndices(raw json.RawMessage) []SnapshotIndex {
	var rows []struct {
		Index string `json:"index"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return []SnapshotIndex{}
	}
	out := make([]SnapshotIndex, 0, len(rows))
	for _, r := range rows {
		out = append(out, SnapshotIndex{Name: r.Index, Special: startsWithDot(r.Index)})
	}
	return out
}

func startsWithDot(s string) bool { return len(s) > 0 && s[0] == '.' }
