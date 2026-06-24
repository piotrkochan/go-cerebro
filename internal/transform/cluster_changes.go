package transform

import "encoding/json"

// Changes is the payload of the /cluster_changes endpoint.
type Changes struct {
	Indices     []string `json:"indices" doc:"All index names, including closed indices."`
	Nodes       []any    `json:"nodes" doc:"Distinct node names."`
	ClusterName any      `json:"cluster_name" doc:"Cluster name."`
}

// ClusterChanges combines aliases, transport nodes, and blocks into a {indices, nodes, cluster_name} payload.
// Port of services/cluster_changes/ClusterChangesDataService.scala.
func ClusterChanges(aliases, nodesInfo, state json.RawMessage) (Changes, error) {
	var aliasesMap map[string]json.RawMessage
	_ = json.Unmarshal(aliases, &aliasesMap)
	indices := make([]string, 0, len(aliasesMap))
	for k := range aliasesMap {
		indices = append(indices, k)
	}

	var stateObj map[string]any
	_ = json.Unmarshal(state, &stateObj)

	closedIndices := []string{}
	if blocksIdx, ok := getNested(stateObj, "blocks", "indices").(map[string]any); ok {
		for name, blk := range blocksIdx {
			if blkMap, ok := blk.(map[string]any); ok {
				if _, ok := blkMap["4"].(map[string]any); ok {
					closedIndices = append(closedIndices, name)
				}
			}
		}
	}

	var infoNode any
	_ = json.Unmarshal(nodesInfo, &infoNode)
	nodes := []any{}
	seen := map[string]bool{}
	collectByKeyDistinct(infoNode, "name", &nodes, seen)

	return Changes{
		Indices:     append(indices, closedIndices...),
		Nodes:       nodes,
		ClusterName: stateObj["cluster_name"],
	}, nil
}

func collectByKeyDistinct(n any, key string, out *[]any, seen map[string]bool) {
	switch v := n.(type) {
	case map[string]any:
		for k, child := range v {
			if k == key {
				if s, ok := child.(string); ok {
					if !seen[s] {
						seen[s] = true
						*out = append(*out, child)
					}
				} else {
					*out = append(*out, child)
				}
			}
			collectByKeyDistinct(child, key, out, seen)
		}
	case []any:
		for _, child := range v {
			collectByKeyDistinct(child, key, out, seen)
		}
	}
}
