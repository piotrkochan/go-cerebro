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

