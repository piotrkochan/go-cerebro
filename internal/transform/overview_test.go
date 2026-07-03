package transform

import "testing"

func TestDataStreamBackingIndicesSupportsClusterStateShapes(t *testing.T) {
	tests := []struct {
		name  string
		state map[string]any
	}{
		{
			name: "elasticsearch 8 metadata data_stream",
			state: map[string]any{
				"metadata": map[string]any{
					"data_stream": map[string]any{
						"data_stream": map[string]any{
							"logs-app": map[string]any{
								"indices": []any{
									map[string]any{"index_name": ".ds-logs-app-000001"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "data_streams map",
			state: map[string]any{
				"metadata": map[string]any{
					"data_streams": map[string]any{
						"logs-app": map[string]any{
							"indices": []any{
								map[string]any{"index_name": ".ds-logs-app-000001"},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dataStreamBackingIndices(tt.state)
			if got[".ds-logs-app-000001"] != "logs-app" {
				t.Fatalf("expected backing index to map to data stream, got %#v", got)
			}
		})
	}
}

func TestIndexingCompleteIndicesSupportsSettingsShapes(t *testing.T) {
	settings := map[string]any{
		"logs-000001": map[string]any{
			"settings": map[string]any{
				"index": map[string]any{
					"lifecycle": map[string]any{
						"indexing_complete": "true",
					},
				},
			},
		},
		"logs-000002": map[string]any{
			"settings.index.lifecycle.indexing_complete": true,
		},
		"logs-000003": map[string]any{
			"index.lifecycle.indexing_complete": "false",
		},
	}

	got := indexingCompleteIndices(settings)
	if !got["logs-000001"] || !got["logs-000002"] {
		t.Fatalf("expected completed indices to be detected, got %#v", got)
	}
	if got["logs-000003"] {
		t.Fatalf("expected false setting to stay incomplete, got %#v", got)
	}
}

func TestBuildIndicesMarksIndexingComplete(t *testing.T) {
	state := map[string]any{
		"routing_table": map[string]any{
			"indices": map[string]any{
				"logs-000001": map[string]any{
					"shards": map[string]any{},
				},
			},
		},
	}

	got := buildIndices(state, map[string]any{}, map[string]any{}, map[string]bool{"logs-000001": true})
	if len(got) != 1 {
		t.Fatalf("expected one index, got %#v", got)
	}
	if !got[0].IndexingComplete {
		t.Fatalf("expected index to be marked indexing complete, got %#v", got[0])
	}
}
