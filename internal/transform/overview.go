package transform

import (
	"encoding/json"
	"strconv"
)

// Overview is the payload of the /overview endpoint — the cluster dashboard data.
type Overview struct {
	ClusterName         any             `json:"cluster_name" doc:"Cluster name."`
	Status              any             `json:"status" doc:"Cluster health status (green/yellow/red)."`
	NumberOfNodes       any             `json:"number_of_nodes" doc:"Number of nodes in the cluster."`
	ActivePrimaryShards any             `json:"active_primary_shards"`
	ActiveShards        any             `json:"active_shards"`
	RelocatingShards    any             `json:"relocating_shards"`
	InitializingShards  any             `json:"initializing_shards"`
	UnassignedShards    any             `json:"unassigned_shards"`
	DocsCount           any             `json:"docs_count" doc:"Total primary docs count."`
	SizeInBytes         any             `json:"size_in_bytes" doc:"Total store size in bytes."`
	TotalIndices        int             `json:"total_indices"`
	ClosedIndices       int             `json:"closed_indices"`
	SpecialIndices      int             `json:"special_indices" doc:"Number of dot-prefixed (system) indices."`
	Indices             []OverviewIndex `json:"indices"`
	Nodes               []OverviewNode  `json:"nodes"`
	ShardAllocation     bool            `json:"shard_allocation" doc:"Whether shard allocation is enabled (cluster.routing.allocation.enable == all)."`
}

// OverviewIndex is one index entry on the cluster dashboard. For closed indices
// reported via cluster blocks (ES < 7) only name/closed/special are populated.
type OverviewIndex struct {
	Name             string                      `json:"name" doc:"Index name."`
	Closed           bool                        `json:"closed"`
	Special          bool                        `json:"special" doc:"Whether the index is dot-prefixed (system index)."`
	DataStream       string                      `json:"data_stream,omitempty" doc:"Data stream name when this index is a backing index."`
	Unhealthy        bool                        `json:"unhealthy" doc:"Whether any shard is not STARTED."`
	DocCount         any                         `json:"doc_count"`
	DeletedDocs      any                         `json:"deleted_docs"`
	SizeInBytes      any                         `json:"size_in_bytes" doc:"Primary store size in bytes."`
	TotalSizeInBytes any                         `json:"total_size_in_bytes"`
	Aliases          []string                    `json:"aliases"`
	NumShards        int                         `json:"num_shards"`
	NumReplicas      int                         `json:"num_replicas"`
	Shards           map[string][]map[string]any `json:"shards" doc:"Routing table shards grouped by node name ('unassigned' for unassigned shards)."`
}

// OverviewHeap is the JVM heap section of a dashboard node.
type OverviewHeap struct {
	Used        any `json:"used" doc:"Used JVM heap in bytes."`
	Committed   any `json:"committed" doc:"Committed JVM heap in bytes."`
	UsedPercent any `json:"used_percent"`
	Max         any `json:"max" doc:"Maximum JVM heap in bytes."`
}

// OverviewDisk is the filesystem section of a dashboard node.
type OverviewDisk struct {
	Total       float64 `json:"total" doc:"Total filesystem size in bytes."`
	Free        float64 `json:"free" doc:"Free filesystem size in bytes."`
	UsedPercent int     `json:"used_percent"`
}

// OverviewNode is one node entry on the cluster dashboard.
type OverviewNode struct {
	ID                  string         `json:"id" doc:"Node id."`
	CurrentMaster       bool           `json:"current_master" doc:"Whether this node is the elected master."`
	Name                any            `json:"name" doc:"Node name."`
	Host                any            `json:"host"`
	IP                  any            `json:"ip"`
	ESVersion           any            `json:"es_version" doc:"Elasticsearch version."`
	JVMVersion          any            `json:"jvm_version"`
	LoadAverage         any            `json:"load_average" doc:"1-minute load average."`
	AvailableProcessors any            `json:"available_processors"`
	CPUPercent          any            `json:"cpu_percent"`
	Master              bool           `json:"master" doc:"Whether the node is master-eligible."`
	Data                bool           `json:"data" doc:"Whether the node has a data role."`
	Coordinating        bool           `json:"coordinating" doc:"Whether the node is a coordinating-only node."`
	Ingest              bool           `json:"ingest" doc:"Whether the node has the ingest role."`
	Heap                OverviewHeap   `json:"heap"`
	Disk                OverviewDisk   `json:"disk"`
	Attributes          map[string]any `json:"attributes" doc:"Custom node attributes (internal ES attributes filtered out)."`
}

// ClusterOverview merges responses from multiple ES endpoints into the single payload consumed by the UI.
// Port of services/overview/OverviewDataService.scala + models/overview/{ClusterOverview,Index,ClosedIndex,Node}.scala.
func ClusterOverview(clusterState, nodesStats, indicesStats, clusterSettings, aliases, clusterHealth, nodesInfo json.RawMessage) (Overview, error) {
	var state map[string]any
	_ = json.Unmarshal(clusterState, &state)
	var settings map[string]any
	_ = json.Unmarshal(clusterSettings, &settings)
	var iStats map[string]any
	_ = json.Unmarshal(indicesStats, &iStats)
	var aliasesMap map[string]any
	_ = json.Unmarshal(aliases, &aliasesMap)
	var health map[string]any
	_ = json.Unmarshal(clusterHealth, &health)
	var nStats map[string]any
	_ = json.Unmarshal(nodesStats, &nStats)
	var nInfo map[string]any
	_ = json.Unmarshal(nodesInfo, &nInfo)

	masterNodeID, _ := state["master_node"].(string)

	persistent := settingsAlloc(settings, "persistent")
	transient := settingsAlloc(settings, "transient")
	var alloc string
	if transient != "" {
		alloc = transient
	} else if persistent != "" {
		alloc = persistent
	} else {
		alloc = "all"
	}

	indices := buildIndices(state, iStats, aliasesMap)
	closedCount := 0
	specialCount := 0
	for _, idx := range indices {
		if idx.Closed {
			closedCount++
		}
		if idx.Special {
			specialCount++
		}
	}

	docsCount := getNested(iStats, "_all", "primaries", "docs", "count")
	if docsCount == nil {
		docsCount = float64(0)
	}
	sizeBytes := getNested(iStats, "_all", "total", "store", "size_in_bytes")
	if sizeBytes == nil {
		sizeBytes = float64(0)
	}

	return Overview{
		ClusterName:         health["cluster_name"],
		Status:              health["status"],
		NumberOfNodes:       health["number_of_nodes"],
		ActivePrimaryShards: health["active_primary_shards"],
		ActiveShards:        health["active_shards"],
		RelocatingShards:    health["relocating_shards"],
		InitializingShards:  health["initializing_shards"],
		UnassignedShards:    health["unassigned_shards"],
		DocsCount:           docsCount,
		SizeInBytes:         sizeBytes,
		TotalIndices:        len(indices),
		ClosedIndices:       closedCount,
		SpecialIndices:      specialCount,
		Indices:             indices,
		Nodes:               buildOverviewNodes(masterNodeID, nInfo, nStats),
		ShardAllocation:     alloc == "all",
	}, nil
}

func settingsAlloc(settings map[string]any, key string) string {
	v := getNested(settings, key, "cluster", "routing", "allocation", "enable")
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func buildOverviewNodes(masterID string, nodesInfo, nodesStats map[string]any) []OverviewNode {
	out := []OverviewNode{}
	statsNodes, _ := nodesStats["nodes"].(map[string]any)
	infoNodes, _ := nodesInfo["nodes"].(map[string]any)
	for id, st := range statsNodes {
		stats, _ := st.(map[string]any)
		info, _ := infoNodes[id].(map[string]any)
		out = append(out, overviewNode(id, info, stats, masterID))
	}
	return out
}

func overviewNode(id string, info, stats map[string]any, masterID string) OverviewNode {
	if info == nil {
		info = map[string]any{}
	}
	roles := parseNodeRoles(stats)

	totalBytes, _ := getNumber(stats, "fs", "total", "total_in_bytes")
	freeBytes, _ := getNumber(stats, "fs", "total", "free_in_bytes")
	usedPercent := 0
	if totalBytes > 0 {
		usedPercent = 100 - int(100*(freeBytes/totalBytes))
	}

	loadAvg := getNested(stats, "os", "cpu", "load_average", "1m")
	if loadAvg == nil {
		loadAvg = getNested(stats, "os", "load_average")
	}
	if loadAvg == nil {
		loadAvg = float64(0)
	}
	cpuPct := getNested(stats, "os", "cpu", "percent")
	if cpuPct == nil {
		cpuPct = getNested(stats, "os", "cpu_percent")
	}
	if cpuPct == nil {
		cpuPct = 0
	}

	return OverviewNode{
		ID:                  id,
		CurrentMaster:       id == masterID,
		Name:                stats["name"],
		Host:                stats["host"],
		IP:                  stats["ip"],
		ESVersion:           info["version"],
		JVMVersion:          getNested(info, "jvm", "version"),
		LoadAverage:         loadAvg,
		AvailableProcessors: getNested(info, "os", "available_processors"),
		CPUPercent:          cpuPct,
		Master:              roles.master,
		Data:                roles.data,
		Coordinating:        roles.coordinating(),
		Ingest:              roles.ingest,
		Heap: OverviewHeap{
			Used:        getNested(stats, "jvm", "mem", "heap_used_in_bytes"),
			Committed:   getNested(stats, "jvm", "mem", "heap_committed_in_bytes"),
			UsedPercent: getNested(stats, "jvm", "mem", "heap_used_percent"),
			Max:         getNested(stats, "jvm", "mem", "heap_max_in_bytes"),
		},
		Disk: OverviewDisk{
			Total:       totalBytes,
			Free:        freeBytes,
			UsedPercent: usedPercent,
		},
		Attributes: nodeAttrs(info),
	}
}

func buildIndices(state, indicesStats, aliases map[string]any) []OverviewIndex {
	routing, _ := getNested(state, "routing_table", "indices").(map[string]any)
	blocks, _ := getNested(state, "blocks", "indices").(map[string]any)
	if blocks == nil {
		blocks = map[string]any{}
	}
	stats, _ := indicesStats["indices"].(map[string]any)
	if stats == nil {
		stats = map[string]any{}
	}
	dataStreams := dataStreamBackingIndices(state)

	out := []OverviewIndex{}
	seen := map[string]bool{}
	for index, shardsAny := range routing {
		shards, _ := shardsAny.(map[string]any)
		seen[index] = true
		idxStats, _ := stats[index].(map[string]any)
		idxBlock, _ := blocks[index].(map[string]any)
		idxAliases := map[string]any{}
		if a, ok := aliases[index].(map[string]any); ok {
			if al, ok := a["aliases"].(map[string]any); ok {
				idxAliases = al
			}
		}
		out = append(out, buildIndex(index, idxStats, shards, idxAliases, idxBlock, dataStreams[index]))
	}
	// Closed indices (ES < 7.X)
	for name, blk := range blocks {
		if seen[name] {
			continue
		}
		blkMap, _ := blk.(map[string]any)
		if _, ok := blkMap["4"]; ok {
			out = append(out, OverviewIndex{
				Name:       name,
				Closed:     true,
				Special:    startsWithDot(name),
				DataStream: dataStreams[name],
			})
		}
	}
	return out
}

func buildIndex(name string, stats, routing, aliasesObj, indexBlock map[string]any, dataStream string) OverviewIndex {
	shardMap := createShardMap(routing)
	docCount := getNested(stats, "primaries", "docs", "count")
	if docCount == nil {
		docCount = float64(0)
	}
	deleted := getNested(stats, "primaries", "docs", "deleted")
	if deleted == nil {
		deleted = float64(0)
	}
	primarySize := getNested(stats, "primaries", "store", "size_in_bytes")
	if primarySize == nil {
		primarySize = float64(0)
	}
	totalSize := getNested(stats, "total", "store", "size_in_bytes")
	if totalSize == nil {
		totalSize = float64(0)
	}

	aliasesArr := []string{}
	for k := range aliasesObj {
		aliasesArr = append(aliasesArr, k)
	}

	numShards := 0
	numReplicas := 0
	if rs, ok := routing["shards"].(map[string]any); ok {
		for k := range rs {
			n, _ := strconv.Atoi(k)
			if n+1 > numShards {
				numShards = n + 1
			}
		}
		if first, ok := rs["0"].([]any); ok {
			numReplicas = len(first) - 1
		}
	}

	closed := false
	if indexBlock != nil {
		if _, ok := indexBlock["4"]; ok {
			closed = true
		}
	}

	return OverviewIndex{
		Name:             name,
		Closed:           closed,
		Special:          startsWithDot(name),
		DataStream:       dataStream,
		Unhealthy:        isIndexUnhealthy(shardMap),
		DocCount:         docCount,
		DeletedDocs:      deleted,
		SizeInBytes:      primarySize,
		TotalSizeInBytes: totalSize,
		Aliases:          aliasesArr,
		NumShards:        numShards,
		NumReplicas:      numReplicas,
		Shards:           shardMap,
	}
}

func dataStreamBackingIndices(state map[string]any) map[string]string {
	out := map[string]string{}
	streams, _ := getNested(state, "metadata", "data_streams").(map[string]any)
	if len(streams) == 0 {
		streams, _ = getNested(state, "metadata", "data_stream", "data_stream").(map[string]any)
	}
	if len(streams) == 0 {
		streams, _ = getNested(state, "metadata", "data_stream").(map[string]any)
	}
	for streamName, raw := range streams {
		if streamName == "data_stream_aliases" {
			continue
		}
		stream, _ := raw.(map[string]any)
		indices, _ := stream["indices"].([]any)
		for _, rawIndex := range indices {
			index, _ := rawIndex.(map[string]any)
			indexName, _ := index["index_name"].(string)
			if indexName == "" {
				indexName, _ = index["name"].(string)
			}
			if indexName != "" {
				out[indexName] = streamName
			}
		}
	}
	return out
}

func createShardMap(routing map[string]any) map[string][]map[string]any {
	out := map[string][]map[string]any{}
	rs, _ := routing["shards"].(map[string]any)
	for _, shardsAny := range rs {
		shards, _ := shardsAny.([]any)
		for _, s := range shards {
			shard, _ := s.(map[string]any)
			node, _ := shard["node"].(string)
			if node == "" {
				node = "unassigned"
			}
			out[node] = append(out[node], shard)
			if relocating, ok := shard["relocating_node"].(string); ok && relocating != "" {
				out[relocating] = append(out[relocating], map[string]any{
					"node":    relocating,
					"index":   shard["index"],
					"state":   "INITIALIZING",
					"shard":   shard["shard"],
					"primary": false,
				})
			}
		}
	}
	return out
}

func isIndexUnhealthy(shardMap map[string][]map[string]any) bool {
	for _, shards := range shardMap {
		for _, shard := range shards {
			if state, _ := shard["state"].(string); state != "STARTED" {
				return true
			}
		}
	}
	return false
}
