package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnassignedShardsFiltersAndNormalizesRows(t *testing.T) {
	raw := json.RawMessage(`[
		{"index":"logs","shard":0,"prirep":"r","state":"UNASSIGNED","unassigned.reason":"INDEX_REOPENED"},
		{"index":"users","shard":"1","prirep":"p","state":"STARTED"},
		{"index":"archive","shard":"2","prirep":"r","state":"unassigned","unassigned.reason.keyword":"NODE_LEFT"}
	]`)

	got := unassignedShards(raw)

	require.Len(t, got, 2)
	assert.Equal(t, navbarUnassignedShard{
		Index:          "logs",
		Shard:          "0",
		PrimaryReplica: "r",
		Reason:         "INDEX_REOPENED",
	}, got[0])
	assert.Equal(t, navbarUnassignedShard{
		Index:          "archive",
		Shard:          "2",
		PrimaryReplica: "r",
		Reason:         "NODE_LEFT",
	}, got[1])
}

func TestParseAllocationExplanationCollectsBlockingDeciders(t *testing.T) {
	raw := json.RawMessage(`{
		"can_allocate": "no",
		"node_allocation_decisions": [
			{"deciders": [
				{"decider":"same_shard","decision":"NO","explanation":"same shard on node"},
				{"decider":"filter","decision":"YES","explanation":"allowed"}
			]},
			{"deciders": [
				{"decider":"same_shard","decision":"NO","explanation":"duplicate should be ignored"},
				{"decider":"disk_threshold","decision":"NO","explanation":"disk is too high"}
			]}
		]
	}`)

	got := parseAllocationExplanation(raw)

	assert.Equal(t, "no", got.Decision)
	assert.Equal(t, "same shard on node", got.Explanation)
	assert.Equal(t, []string{"same_shard", "disk_threshold"}, got.Deciders)
	assert.True(t, got.HasDecider("same_shard"))
	assert.False(t, got.HasDecider("awareness"))
}

func TestRawJSONHelpers(t *testing.T) {
	assert.Equal(t, "yellow", rawJSONText(json.RawMessage(`"yellow"`)))
	assert.Equal(t, "yellow", rawJSONText(json.RawMessage(`yellow`)))
	assert.Equal(t, 7, rawJSONInt(json.RawMessage(`7`)))
	assert.Equal(t, 7, rawJSONInt(json.RawMessage(`7.9`)))
	assert.Equal(t, 0, rawJSONInt(json.RawMessage(`"bad"`)))
}

func TestPluralize(t *testing.T) {
	assert.Equal(t, "1 unassigned shard", pluralize(1, "unassigned shard"))
	assert.Equal(t, "2 unassigned shards", pluralize(2, "unassigned shard"))
}
