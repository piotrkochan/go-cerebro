package transform

import (
	"encoding/json"
	"strconv"
)

type DataStreamsResult struct {
	Supported bool         `json:"supported" doc:"Whether the target Elasticsearch supports data streams."`
	Items     []DataStream `json:"items" doc:"Data streams."`
}

type DataStream struct {
	Name                    string                   `json:"name"`
	Status                  string                   `json:"status,omitempty"`
	TimestampField          string                   `json:"timestamp_field,omitempty"`
	Template                string                   `json:"template,omitempty"`
	ManagedBy               string                   `json:"managed_by,omitempty"`
	NextGenerationManagedBy string                   `json:"next_generation_managed_by,omitempty"`
	Generation              int                      `json:"generation"`
	BackingIndicesCount     int                      `json:"backing_indices_count"`
	StoreSizeBytes          float64                  `json:"store_size_bytes"`
	MaximumTimestamp        any                      `json:"maximum_timestamp,omitempty"`
	Hidden                  bool                     `json:"hidden"`
	System                  bool                     `json:"system"`
	PreferILM               bool                     `json:"prefer_ilm"`
	RolloverOnWrite         bool                     `json:"rollover_on_write"`
	Lifecycle               json.RawMessage          `json:"lifecycle,omitempty"`
	BackingIndices          []DataStreamBackingIndex `json:"backing_indices"`
}

type DataStreamBackingIndex struct {
	Name           string `json:"name"`
	UUID           string `json:"uuid,omitempty"`
	WriteIndex     bool   `json:"write_index"`
	ManagedBy      string `json:"managed_by,omitempty"`
	Health         string `json:"health,omitempty"`
	Status         string `json:"status,omitempty"`
	DocsCount      any    `json:"docs_count,omitempty"`
	StoreSizeBytes any    `json:"store_size_bytes,omitempty"`
	ILMManaged     bool   `json:"ilm_managed"`
	ILMPolicy      string `json:"ilm_policy,omitempty"`
	ILMPhase       string `json:"ilm_phase,omitempty"`
	ILMAction      string `json:"ilm_action,omitempty"`
	ILMStep        string `json:"ilm_step,omitempty"`
	ILMError       string `json:"ilm_error,omitempty"`
}

func DataStreamsUnsupported() DataStreamsResult {
	return DataStreamsResult{Supported: false, Items: []DataStream{}}
}

func DataStreamSummaries(dataStreamsRaw json.RawMessage) DataStreamsResult {
	result := DataStreams(dataStreamsRaw, nil, nil, nil)
	for i := range result.Items {
		result.Items[i].BackingIndices = nil
		result.Items[i].Lifecycle = nil
		result.Items[i].StoreSizeBytes = 0
		result.Items[i].MaximumTimestamp = nil
	}
	return result
}

func DataStreamDetail(dataStreamsRaw, statsRaw, catRaw, ilmRaw json.RawMessage, name string) (DataStream, bool) {
	result := DataStreams(dataStreamsRaw, statsRaw, catRaw, ilmRaw)
	for _, stream := range result.Items {
		if stream.Name == name {
			return stream, true
		}
	}
	return DataStream{}, false
}

func DataStreams(dataStreamsRaw, statsRaw, catRaw, ilmRaw json.RawMessage) DataStreamsResult {
	var root struct {
		DataStreams []struct {
			Name           string `json:"name"`
			TimestampField struct {
				Name string `json:"name"`
			} `json:"timestamp_field"`
			Indices []struct {
				IndexName string `json:"index_name"`
				IndexUUID string `json:"index_uuid"`
				ManagedBy string `json:"managed_by"`
			} `json:"indices"`
			Generation              int             `json:"generation"`
			Status                  string          `json:"status"`
			Template                string          `json:"template"`
			ManagedBy               string          `json:"managed_by"`
			NextGenerationManagedBy string          `json:"next_generation_managed_by"`
			PreferILM               bool            `json:"prefer_ilm"`
			Hidden                  bool            `json:"hidden"`
			System                  bool            `json:"system"`
			RolloverOnWrite         bool            `json:"rollover_on_write"`
			Lifecycle               json.RawMessage `json:"lifecycle"`
		} `json:"data_streams"`
	}
	if err := json.Unmarshal(dataStreamsRaw, &root); err != nil {
		return DataStreamsUnsupported()
	}

	stats := dataStreamStats(statsRaw)
	cat := dataStreamCatIndices(catRaw)
	ilm := dataStreamILM(ilmRaw)
	out := DataStreamsResult{Supported: true, Items: make([]DataStream, 0, len(root.DataStreams))}
	for _, stream := range root.DataStreams {
		item := DataStream{
			Name:                    stream.Name,
			Status:                  stream.Status,
			TimestampField:          stream.TimestampField.Name,
			Template:                stream.Template,
			ManagedBy:               stream.ManagedBy,
			NextGenerationManagedBy: stream.NextGenerationManagedBy,
			Generation:              stream.Generation,
			BackingIndicesCount:     len(stream.Indices),
			Hidden:                  stream.Hidden,
			System:                  stream.System,
			PreferILM:               stream.PreferILM,
			RolloverOnWrite:         stream.RolloverOnWrite,
			Lifecycle:               stream.Lifecycle,
			BackingIndices:          []DataStreamBackingIndex{},
		}
		if st, ok := stats[stream.Name]; ok {
			item.BackingIndicesCount = st.BackingIndices
			item.StoreSizeBytes = st.StoreSizeBytes
			item.MaximumTimestamp = st.MaximumTimestamp
		}
		last := len(stream.Indices) - 1
		for i, index := range stream.Indices {
			backing := DataStreamBackingIndex{
				Name:       index.IndexName,
				UUID:       index.IndexUUID,
				WriteIndex: i == last,
				ManagedBy:  index.ManagedBy,
			}
			if c, ok := cat[index.IndexName]; ok {
				backing.Health = c.Health
				backing.Status = c.Status
				backing.DocsCount = c.DocsCount
				backing.StoreSizeBytes = c.StoreSizeBytes
			}
			if info, ok := ilm[index.IndexName]; ok {
				backing.ILMManaged = info.Managed
				backing.ILMPolicy = info.Policy
				backing.ILMPhase = info.Phase
				backing.ILMAction = info.Action
				backing.ILMStep = info.Step
				backing.ILMError = info.Error
			}
			item.BackingIndices = append(item.BackingIndices, backing)
		}
		out.Items = append(out.Items, item)
	}
	return out
}

type dsStatsEntry struct {
	BackingIndices   int
	StoreSizeBytes   float64
	MaximumTimestamp any
}

func dataStreamStats(raw json.RawMessage) map[string]dsStatsEntry {
	var root struct {
		DataStreams []struct {
			Name             string `json:"data_stream"`
			BackingIndices   int    `json:"backing_indices"`
			StoreSizeBytes   any    `json:"store_size_bytes"`
			MaximumTimestamp any    `json:"maximum_timestamp"`
		} `json:"data_streams"`
	}
	_ = json.Unmarshal(raw, &root)
	out := map[string]dsStatsEntry{}
	for _, stream := range root.DataStreams {
		out[stream.Name] = dsStatsEntry{
			BackingIndices:   stream.BackingIndices,
			StoreSizeBytes:   numberFromAny(stream.StoreSizeBytes),
			MaximumTimestamp: stream.MaximumTimestamp,
		}
	}
	return out
}

type dsCatIndex struct {
	Health         string
	Status         string
	DocsCount      any
	StoreSizeBytes any
}

func dataStreamCatIndices(raw json.RawMessage) map[string]dsCatIndex {
	var rows []map[string]any
	_ = json.Unmarshal(raw, &rows)
	out := map[string]dsCatIndex{}
	for _, row := range rows {
		name, _ := row["index"].(string)
		if name == "" {
			continue
		}
		out[name] = dsCatIndex{
			Health:         stringFromAny(row["health"]),
			Status:         stringFromAny(row["status"]),
			DocsCount:      row["docs.count"],
			StoreSizeBytes: row["store.size"],
		}
	}
	return out
}

type dsILMIndex struct {
	Managed bool
	Policy  string
	Phase   string
	Action  string
	Step    string
	Error   string
}

func dataStreamILM(raw json.RawMessage) map[string]dsILMIndex {
	var root struct {
		Indices map[string]struct {
			Managed    bool   `json:"managed"`
			Policy     string `json:"policy"`
			Phase      string `json:"phase"`
			Action     string `json:"action"`
			Step       string `json:"step"`
			StepInfo   any    `json:"step_info"`
			FailedStep string `json:"failed_step"`
		} `json:"indices"`
	}
	_ = json.Unmarshal(raw, &root)
	out := map[string]dsILMIndex{}
	for name, index := range root.Indices {
		out[name] = dsILMIndex{
			Managed: index.Managed,
			Policy:  index.Policy,
			Phase:   index.Phase,
			Action:  index.Action,
			Step:    index.Step,
			Error:   stringFromAny(index.StepInfo),
		}
	}
	return out
}

func numberFromAny(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case string:
		n, _ := strconv.ParseFloat(v, 64)
		return n
	default:
		return 0
	}
}

func stringFromAny(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		raw, _ := json.Marshal(v)
		return string(raw)
	}
}
