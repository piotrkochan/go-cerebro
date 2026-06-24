package transform

import (
	"encoding/json"
	"strconv"
)

// ShardStats returns the matching shard or recovery info for a given (index, node, shard). Port of models/ShardStats.scala.
func ShardStats(index, node string, shard int, stats, recovery json.RawMessage) json.RawMessage {
	if s := pickShardStats(index, node, shard, stats); s != nil {
		return s
	}
	if r := pickShardRecovery(index, node, shard, recovery); r != nil {
		return r
	}
	return json.RawMessage("null")
}

func pickShardStats(index, node string, shard int, raw json.RawMessage) json.RawMessage {
	var root struct {
		Indices map[string]struct {
			Shards map[string][]json.RawMessage `json:"shards"`
		} `json:"indices"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil
	}
	idx, ok := root.Indices[index]
	if !ok {
		return nil
	}
	shards, ok := idx.Shards[strconv.Itoa(shard)]
	if !ok {
		return nil
	}
	for _, s := range shards {
		var routing struct {
			Routing struct {
				Node string `json:"node"`
			} `json:"routing"`
		}
		if err := json.Unmarshal(s, &routing); err == nil && routing.Routing.Node == node {
			return s
		}
	}
	return nil
}

func pickShardRecovery(index, node string, shard int, raw json.RawMessage) json.RawMessage {
	var root map[string]struct {
		Shards []json.RawMessage `json:"shards"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil
	}
	entry, ok := root[index]
	if !ok {
		return nil
	}
	for _, r := range entry.Shards {
		var info struct {
			ID     int `json:"id"`
			Target struct {
				ID string `json:"id"`
			} `json:"target"`
		}
		if err := json.Unmarshal(r, &info); err == nil && info.ID == shard && info.Target.ID == node {
			return r
		}
	}
	return nil
}
