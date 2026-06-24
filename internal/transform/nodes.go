package transform

import (
	"encoding/json"
	"math"
)

var internalNodeAttrs = map[string]bool{
	"ml.machine_memory": true,
	"xpack.installed":   true,
	"transform.node":    true,
	"ml.max_open_jobs":  true,
}

type nodeRoles struct {
	master, data, ingest bool
}

func (r nodeRoles) coordinating() bool { return !r.master && !r.data && !r.ingest }

func parseNodeRoles(info map[string]any) nodeRoles {
	if rolesAny, ok := info["roles"].([]any); ok { // >= 5.X
		dataRoles := map[string]bool{
			"data": true, "data_content": true, "data_hot": true, "data_warm": true, "data_cold": true,
		}
		var r nodeRoles
		for _, role := range rolesAny {
			if s, ok := role.(string); ok {
				if s == "master" {
					r.master = true
				}
				if s == "ingest" {
					r.ingest = true
				}
				if dataRoles[s] {
					r.data = true
				}
			}
		}
		return r
	}
	// 2.X — read attributes
	attrs, _ := info["attributes"].(map[string]any)
	truthy := func(s string) bool { return s == "true" || s == "yes" }
	read := func(key, def string) string {
		if attrs == nil {
			return def
		}
		if v, ok := attrs[key].(string); ok {
			return v
		}
		return def
	}
	master := truthy(read("master", "true"))
	data := truthy(read("data", "true"))
	client := truthy(read("client", "false"))
	return nodeRoles{master: master && !client, data: data && !client, ingest: false}
}

func nodeAttrs(info map[string]any) map[string]any {
	attrs, _ := info["attributes"].(map[string]any)
	out := map[string]any{}
	if attrs == nil {
		return out
	}
	for k, v := range attrs {
		if !internalNodeAttrs[k] {
			out[k] = v
		}
	}
	return out
}

// NodeHeap is the JVM heap section of a node. Values come straight from nodes
// stats and may be human-readable strings or numbers depending on the ES version.
type NodeHeap struct {
	Max     any `json:"max" doc:"Maximum JVM heap."`
	Used    any `json:"used" doc:"Used JVM heap."`
	Percent any `json:"percent" doc:"Used JVM heap percentage."`
}

// NodeDisk is the filesystem usage section of a node.
type NodeDisk struct {
	Total     float64 `json:"total" doc:"Total filesystem size in bytes."`
	Available float64 `json:"available" doc:"Available filesystem size in bytes."`
	Percent   float64 `json:"percent" doc:"Used disk percentage."`
}

// NodeCPU is the CPU usage section of a node.
type NodeCPU struct {
	Process any `json:"process" doc:"Process CPU usage percentage."`
	OS      any `json:"os" doc:"OS CPU usage percentage."`
	Load    any `json:"load" doc:"1-minute load average."`
}

// Node is one entry of the /nodes endpoint payload.
type Node struct {
	ID            string         `json:"id" doc:"Node id."`
	CurrentMaster bool           `json:"current_master" doc:"Whether this node is the elected master."`
	Name          any            `json:"name" doc:"Node name."`
	Host          any            `json:"host" doc:"Node host."`
	Heap          NodeHeap       `json:"heap"`
	Disk          *NodeDisk      `json:"disk" doc:"Disk usage (null when filesystem stats are unavailable)."`
	CPU           NodeCPU        `json:"cpu"`
	Uptime        any            `json:"uptime" doc:"JVM uptime in milliseconds."`
	JVM           any            `json:"jvm" doc:"JVM version."`
	Attributes    map[string]any `json:"attributes" doc:"Custom node attributes (internal ES attributes filtered out)."`
	Version       any            `json:"version" doc:"Elasticsearch version."`
	Master        bool           `json:"master" doc:"Whether the node is master-eligible."`
	Coordinating  bool           `json:"coordinating" doc:"Whether the node is a coordinating-only node."`
	Ingest        bool           `json:"ingest" doc:"Whether the node has the ingest role."`
	Data          bool           `json:"data" doc:"Whether the node has a data role."`
}

// commonsNode builds one Node entry — port of models/nodes/Node.scala + Nodes.scala.
func commonsNode(id string, currentMaster bool, info, stats map[string]any) Node {
	roles := parseNodeRoles(info)

	cpuLoad := getNested(stats, "os", "cpu", "load_average", "1m")
	if cpuLoad == nil {
		cpuLoad = getNested(stats, "os", "load_average")
	}
	osCPU := getNested(stats, "os", "cpu", "percent")
	if osCPU == nil {
		osCPU = getNested(stats, "os", "cpu_percent")
	}

	var disk *NodeDisk
	totalBytes, totalOK := getNumber(stats, "fs", "total", "total_in_bytes")
	availBytes, availOK := getNumber(stats, "fs", "total", "available_in_bytes")
	if totalOK && availOK && totalBytes > 0 {
		disk = &NodeDisk{
			Total:     totalBytes,
			Available: availBytes,
			Percent:   math.Round((1 - (availBytes / totalBytes)) * 100),
		}
	}

	return Node{
		ID:            id,
		CurrentMaster: currentMaster,
		Name:          stats["name"],
		Host:          stats["host"],
		Heap: NodeHeap{
			Max:     getNested(stats, "jvm", "mem", "heap_max"),
			Used:    getNested(stats, "jvm", "mem", "heap_used"),
			Percent: getNested(stats, "jvm", "mem", "heap_used_percent"),
		},
		Disk: disk,
		CPU: NodeCPU{
			Process: getNested(stats, "process", "cpu", "percent"),
			OS:      osCPU,
			Load:    cpuLoad,
		},
		Uptime:       getNested(stats, "jvm", "uptime_in_millis"),
		JVM:          getNested(info, "jvm", "version"),
		Attributes:   nodeAttrs(info),
		Version:      info["version"],
		Master:       roles.master,
		Coordinating: roles.coordinating(),
		Ingest:       roles.ingest,
		Data:         roles.data,
	}
}

// Nodes is the response transformer for /nodes endpoint.
func Nodes(info, stats, master json.RawMessage) []Node {
	var infoNode any
	_ = json.Unmarshal(info, &infoNode)
	var statsNode any
	_ = json.Unmarshal(stats, &statsNode)

	masterID := ""
	{
		var arr []any
		if err := json.Unmarshal(master, &arr); err == nil && len(arr) > 0 {
			if m, ok := arr[0].(map[string]any); ok {
				if v, ok := m["id"].(string); ok {
					masterID = v
				}
			}
		}
	}

	infoMap, _ := infoNode.(map[string]any)
	statsMap, _ := statsNode.(map[string]any)
	infoNodes, _ := infoMap["nodes"].(map[string]any)
	statsNodes, _ := statsMap["nodes"].(map[string]any)

	out := []Node{}
	for id, ni := range infoNodes {
		nodeInfo, _ := ni.(map[string]any)
		nodeStats, _ := statsNodes[id].(map[string]any)
		out = append(out, commonsNode(id, id == masterID, nodeInfo, nodeStats))
	}
	return out
}
