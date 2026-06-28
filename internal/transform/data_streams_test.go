package transform

import (
	"encoding/json"
	"testing"
)

func TestDataStreamsCombinesStatsCatAndILM(t *testing.T) {
	streams := json.RawMessage(`{
		"data_streams": [{
			"name": "logs-app",
			"timestamp_field": {"name": "@timestamp"},
			"indices": [{"index_name": ".ds-logs-app-000001", "index_uuid": "uuid-1", "managed_by": "Unmanaged"}],
			"generation": 1,
			"status": "GREEN",
			"template": "logs-template",
			"prefer_ilm": true,
			"hidden": false,
			"system": false,
			"rollover_on_write": false,
			"lifecycle": {"data_retention": "30d"}
		}]
	}`)
	stats := json.RawMessage(`{
		"data_streams": [{
			"data_stream": "logs-app",
			"backing_indices": 1,
			"store_size_bytes": 1024,
			"maximum_timestamp": 1782554520000
		}]
	}`)
	cat := json.RawMessage(`[
		{"index": ".ds-logs-app-000001", "health": "green", "status": "open", "docs.count": "3", "store.size": "1024"}
	]`)
	ilm := json.RawMessage(`{
		"indices": {
			".ds-logs-app-000001": {"managed": false}
		}
	}`)

	got := DataStreams(streams, stats, cat, ilm)
	if !got.Supported || len(got.Items) != 1 {
		t.Fatalf("expected one supported data stream, got %#v", got)
	}
	stream := got.Items[0]
	if stream.Name != "logs-app" || stream.Template != "logs-template" || stream.StoreSizeBytes != 1024 {
		t.Fatalf("unexpected stream: %#v", stream)
	}
	if len(stream.BackingIndices) != 1 || !stream.BackingIndices[0].WriteIndex || stream.BackingIndices[0].Health != "green" {
		t.Fatalf("unexpected backing index: %#v", stream.BackingIndices)
	}
}
