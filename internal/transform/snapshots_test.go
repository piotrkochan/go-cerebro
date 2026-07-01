package transform

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSnapshotsList(t *testing.T) {
	raw := json.RawMessage(`{"snapshots":[{"snapshot":"daily","state":"SUCCESS"}]}`)

	assert.JSONEq(t, `[{"snapshot":"daily","state":"SUCCESS"}]`, string(SnapshotsList(raw)))
	assert.JSONEq(t, `[]`, string(SnapshotsList(json.RawMessage(`{"repositories":[]}`))))
	assert.JSONEq(t, `[]`, string(SnapshotsList(json.RawMessage(`{`))))
}

func TestSnapshotRepositories(t *testing.T) {
	raw := json.RawMessage(`{"fs-repo":{"type":"fs"},"s3-repo":{"type":"s3"}}`)

	assert.ElementsMatch(t, []string{"fs-repo", "s3-repo"}, SnapshotRepositories(raw))
	assert.Empty(t, SnapshotRepositories(json.RawMessage(`{`)))
}

func TestSnapshotIndicesMarksSpecialIndices(t *testing.T) {
	raw := json.RawMessage(`[
		{"index":"logs"},
		{"index":".security"}
	]`)

	assert.Equal(t, []SnapshotIndex{
		{Name: "logs"},
		{Name: ".security", Special: true},
	}, SnapshotIndices(raw))
	assert.Empty(t, SnapshotIndices(json.RawMessage(`{`)))
}
