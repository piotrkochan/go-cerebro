package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/auth"
	"github.com/lmenezes/cerebro/internal/elastic"
)

type NavbarIn struct {
	ClusterPath
}

func (d *Deps) RegisterNavbar(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "navbar",
		Method:      http.MethodGet,
		Path:        "/navbar",
		Summary:     "Get navbar data",
		Description: "Returns the cluster health document (verbatim from _cluster/health), extended with navbar metadata.",
		Tags:        []string{"navbar"},
	}, func(ctx context.Context, in *NavbarIn) (*RawOutput, error) {
		t, err := clusterTarget(ctx)
		if err != nil {
			return failMsg[RawResponse](400, err.Error())
		}
		resp, err := d.Client.ClusterHealth(ctx, t)
		if err != nil {
			return failMsg[RawResponse](500, err.Error())
		}
		if !resp.IsSuccess() {
			return fail[RawResponse](resp.Status, resp.Body)
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(resp.Body, &m); err != nil {
			return raw(resp.Status, resp.Body)
		}
		version := json.RawMessage(nil)
		if versionResp, err := d.Client.Main(ctx, t); err == nil && versionResp.IsSuccess() {
			var info map[string]json.RawMessage
			if err := json.Unmarshal(versionResp.Body, &info); err == nil {
				version = info["version"]
			}
		}
		if user := auth.UserFrom(ctx); user != "" {
			name, _ := json.Marshal(user)
			m["username"] = name
		}
		features, _ := json.Marshal(map[string]bool{"data_explorer": d.Cfg.Features.DataExplorer})
		m["features"] = features
		if len(version) > 0 {
			m["version"] = version
		}
		if issue := d.navbarHealthIssue(ctx, t, m); len(issue) > 0 {
			m["health_issue"] = issue
		}
		newBody, _ := json.Marshal(m)
		return raw(resp.Status, newBody)
	})
}

type navbarHealthIssue struct {
	Status               string                  `json:"status"`
	Summary              string                  `json:"summary"`
	UnassignedShardCount int                     `json:"unassigned_shard_count"`
	UnassignedShards     []navbarUnassignedShard `json:"unassigned_shards"`
	Fixes                []navbarHealthFix       `json:"fixes"`
}

type navbarUnassignedShard struct {
	Index              string   `json:"index"`
	Shard              string   `json:"shard"`
	PrimaryReplica     string   `json:"primary_replica"`
	Reason             string   `json:"reason"`
	AllocationDecision string   `json:"allocation_decision"`
	Explanation        string   `json:"explanation"`
	Deciders           []string `json:"deciders"`
}

type navbarHealthFix struct {
	Action    string `json:"action"`
	Index     string `json:"index"`
	Setting   string `json:"setting"`
	Value     string `json:"value"`
	Summary   string `json:"summary"`
	Rationale string `json:"rationale"`
}

func (d *Deps) navbarHealthIssue(ctx context.Context, t elastic.Server, health map[string]json.RawMessage) json.RawMessage {
	status := strings.ToLower(rawJSONText(health["status"]))
	if status != "yellow" && status != "red" {
		return nil
	}
	unassignedCount := rawJSONInt(health["unassigned_shards"])
	issue := navbarHealthIssue{
		Status:               status,
		Summary:              fmt.Sprintf("cluster health is %s", status),
		UnassignedShardCount: unassignedCount,
		UnassignedShards:     []navbarUnassignedShard{},
		Fixes:                []navbarHealthFix{},
	}
	if unassignedCount > 0 {
		issue.Summary = pluralize(unassignedCount, "unassigned shard")
		columns := []string{"index", "shard", "prirep", "state", "unassigned.reason"}
		if resp, err := d.Client.CatShards(ctx, columns, t); err == nil && resp.IsSuccess() {
			issue.UnassignedShards = unassignedShards(resp.Body)
			issue.explainUnassignedShards(ctx, d.Client, t, rawJSONInt(health["number_of_data_nodes"]))
		}
	}
	body, _ := json.Marshal(issue)
	return body
}

func (issue *navbarHealthIssue) explainUnassignedShards(ctx context.Context, client elastic.Client, t elastic.Server, dataNodes int) {
	safeReplicaFixes := map[string]navbarHealthFix{}
	for i := range issue.UnassignedShards {
		shard := &issue.UnassignedShards[i]
		shardNumber, err := strconv.Atoi(shard.Shard)
		if err != nil {
			continue
		}
		primary := shard.PrimaryReplica == "p"
		resp, err := client.AllocationExplain(ctx, shard.Index, shardNumber, primary, t)
		if err != nil || !resp.IsSuccess() {
			continue
		}
		explanation := parseAllocationExplanation(resp.Body)
		shard.AllocationDecision = explanation.Decision
		shard.Explanation = explanation.Explanation
		shard.Deciders = explanation.Deciders
		if !primary && dataNodes > 0 && explanation.HasDecider("same_shard") {
			targetReplicas := dataNodes - 1
			shard.Explanation = fmt.Sprintf("This replica cannot be allocated because every data node already has a copy of shard %s for index %s. The index replica count is too high for %d data nodes.", shard.Shard, shard.Index, dataNodes)
			safeReplicaFixes[shard.Index] = navbarHealthFix{
				Action:    "set_index_replicas",
				Index:     shard.Index,
				Setting:   "index.number_of_replicas",
				Value:     strconv.Itoa(targetReplicas),
				Summary:   fmt.Sprintf("set replicas on %s to %d", shard.Index, targetReplicas),
				Rationale: fmt.Sprintf("The cluster has %d data nodes. Elasticsearch can place one primary and at most %d replica copy of each shard without putting two copies on the same node.", dataNodes, targetReplicas),
			}
		}
	}
	for _, fix := range safeReplicaFixes {
		issue.Fixes = append(issue.Fixes, fix)
	}
}

type allocationExplanation struct {
	Decision    string
	Explanation string
	Deciders    []string
}

func (e allocationExplanation) HasDecider(name string) bool {
	for _, decider := range e.Deciders {
		if decider == name {
			return true
		}
	}
	return false
}

func parseAllocationExplanation(body json.RawMessage) allocationExplanation {
	var raw struct {
		CanAllocate         string `json:"can_allocate"`
		AllocateExplanation string `json:"allocate_explanation"`
		NodeDecisions       []struct {
			Deciders []struct {
				Decider     string `json:"decider"`
				Decision    string `json:"decision"`
				Explanation string `json:"explanation"`
			} `json:"deciders"`
		} `json:"node_allocation_decisions"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return allocationExplanation{}
	}
	result := allocationExplanation{Decision: raw.CanAllocate, Explanation: raw.AllocateExplanation, Deciders: []string{}}
	seen := map[string]bool{}
	for _, node := range raw.NodeDecisions {
		for _, decider := range node.Deciders {
			if strings.ToUpper(decider.Decision) != "NO" || seen[decider.Decider] {
				continue
			}
			seen[decider.Decider] = true
			result.Deciders = append(result.Deciders, decider.Decider)
			if result.Explanation == "" {
				result.Explanation = decider.Explanation
			}
		}
	}
	if result.Explanation == "" && len(result.Deciders) > 0 {
		result.Explanation = "allocation blocked by " + strings.Join(result.Deciders, ", ")
	}
	return result
}

func unassignedShards(body json.RawMessage) []navbarUnassignedShard {
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		return []navbarUnassignedShard{}
	}
	shards := make([]navbarUnassignedShard, 0)
	for _, row := range rows {
		if strings.ToUpper(mapText(row, "state")) != "UNASSIGNED" {
			continue
		}
		shards = append(shards, navbarUnassignedShard{
			Index:          mapText(row, "index"),
			Shard:          mapText(row, "shard"),
			PrimaryReplica: mapText(row, "prirep"),
			Reason:         firstNonEmpty(mapText(row, "unassigned.reason"), mapText(row, "unassigned.reason.keyword")),
		})
	}
	return shards
}

func rawJSONText(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return strings.Trim(string(raw), `"`)
}

func rawJSONInt(raw json.RawMessage) int {
	var i int
	if err := json.Unmarshal(raw, &i); err == nil {
		return i
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return int(f)
	}
	return 0
}

func mapText(row map[string]any, key string) string {
	switch v := row[key].(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func pluralize(count int, singular string) string {
	if count == 1 {
		return fmt.Sprintf("1 %s", singular)
	}
	return fmt.Sprintf("%d %ss", count, singular)
}
