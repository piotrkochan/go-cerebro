//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lmenezes/cerebro/internal/config"
	"github.com/lmenezes/cerebro/internal/elastic"
	"github.com/stretchr/testify/require"
)

func TestElasticsearchCompatibility(t *testing.T) {
	esURL := strings.TrimSpace(os.Getenv("CEREBRO_E2E_ES_URL"))
	if esURL == "" {
		t.Skip("set CEREBRO_E2E_ES_URL to run Elasticsearch compatibility tests")
	}

	major := envInt(t, "CEREBRO_E2E_ES_MAJOR")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, err := elastic.NewHTTPClientWithConfig(&http.Client{Timeout: 30 * time.Second}, config.ES{
		MaxResponseBytes: config.DefaultMaxResponseBytes,
	})
	require.NoError(t, err)

	target := elastic.Server{Host: config.Host{Name: "e2e", Host: esURL}}
	index := fmt.Sprintf("cerebro-e2e-%d", time.Now().UnixNano())
	alias := index + "-alias"
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_, _ = client.DeleteIndex(cleanupCtx, index, target)
	})

	requireCall(t, "main", func() (elastic.Response, error) { return client.Main(ctx, target) })
	requireCall(t, "cluster health", func() (elastic.Response, error) { return client.ClusterHealth(ctx, target) })
	requireCall(t, "cluster state", func() (elastic.Response, error) { return client.ClusterState(ctx, target) })
	requireCall(t, "cluster settings", func() (elastic.Response, error) { return client.ClusterSettings(ctx, target) })
	requireCall(t, "aliases", func() (elastic.Response, error) { return client.GetAliases(ctx, target) })
	requireCall(t, "cat indices", func() (elastic.Response, error) { return client.GetIndices(ctx, target) })
	requireCall(t, "cat nodes", func() (elastic.Response, error) { return client.GetNodes(ctx, target) })
	requireCall(t, "cat health", func() (elastic.Response, error) { return client.CatRequest(ctx, "health", target) })
	requireCall(t, "cat master", func() (elastic.Response, error) { return client.CatMaster(ctx, target) })
	requireCall(t, "nodes info", func() (elastic.Response, error) {
		return client.Nodes(ctx, []string{"jvm", "os", "process"}, target)
	})
	requireCall(t, "nodes stats", func() (elastic.Response, error) {
		return client.NodesStats(ctx, []string{"jvm", "os", "process"}, target)
	})

	requireCall(t, "create index", func() (elastic.Response, error) {
		return client.CreateIndex(ctx, index, json.RawMessage(`{"settings":{"number_of_shards":1,"number_of_replicas":0}}`), target)
	})
	requireCall(t, "index metadata", func() (elastic.Response, error) { return client.GetIndexMetadata(ctx, index, target) })
	requireCall(t, "save document", func() (elastic.Response, error) {
		return client.SaveIndexDocument(ctx, index, "doc-1", json.RawMessage(`{"title":"compatibility","items_count":7}`), target)
	})
	requireCall(t, "refresh index", func() (elastic.Response, error) { return client.RefreshIndex(ctx, index, target) })
	requireCall(t, "flush index", func() (elastic.Response, error) { return client.FlushIndex(ctx, index, target) })
	requireCall(t, "clear index cache", func() (elastic.Response, error) { return client.ClearIndexCache(ctx, index, target) })
	requireCall(t, "index settings", func() (elastic.Response, error) { return client.GetIndexSettings(ctx, index, target) })
	requireCall(t, "index flat settings", func() (elastic.Response, error) { return client.GetIndexSettingsFlat(ctx, index, target) })
	requireCall(t, "index mapping", func() (elastic.Response, error) { return client.GetIndexMapping(ctx, index, target) })
	requireCall(t, "index stats", func() (elastic.Response, error) { return client.IndexStats(ctx, index, target) })
	requireCall(t, "shard stats", func() (elastic.Response, error) { return client.GetShardStats(ctx, index, target) })
	requireCall(t, "index recovery", func() (elastic.Response, error) { return client.GetIndexRecovery(ctx, index, target) })
	requireCall(t, "analyze by analyzer", func() (elastic.Response, error) {
		return client.AnalyzeTextByAnalyzer(ctx, index, "standard", "compatibility", target)
	})
	requireCall(t, "analyze by field", func() (elastic.Response, error) {
		return client.AnalyzeTextByField(ctx, index, "title", "compatibility", target)
	})
	requireCall(t, "force merge", func() (elastic.Response, error) { return client.ForceMerge(ctx, index, target) })
	searchResp := requireCall(t, "search documents", func() (elastic.Response, error) {
		return client.SearchIndexDocuments(ctx, index, searchBody(major), target)
	})
	requireSearchHit(t, searchResp.Body)
	requireCall(t, "add alias", func() (elastic.Response, error) {
		return client.UpdateAliases(ctx, []json.RawMessage{
			json.RawMessage(fmt.Sprintf(`{"add":{"index":%q,"alias":%q}}`, index, alias)),
		}, target)
	})
	requireCall(t, "cat aliases", func() (elastic.Response, error) { return client.CatRequest(ctx, "aliases", target) })
	requireCall(t, "remove alias", func() (elastic.Response, error) {
		return client.UpdateAliases(ctx, []json.RawMessage{
			json.RawMessage(fmt.Sprintf(`{"remove":{"index":%q,"alias":%q}}`, index, alias)),
		}, target)
	})
	requireCall(t, "close index", func() (elastic.Response, error) { return client.CloseIndex(ctx, index, target) })
	requireCall(t, "open index", func() (elastic.Response, error) { return client.OpenIndex(ctx, index, target) })
	requireCall(t, "delete index", func() (elastic.Response, error) { return client.DeleteIndex(ctx, index, target) })
}

func requireCall(t *testing.T, operation string, call func() (elastic.Response, error)) elastic.Response {
	t.Helper()
	resp, err := call()
	require.NoError(t, err, operation)
	require.Truef(t, resp.IsSuccess(), "%s returned status %d: %s", operation, resp.Status, string(resp.Body))
	return resp
}

func requireSearchHit(t *testing.T, raw json.RawMessage) {
	t.Helper()
	var root struct {
		Hits struct {
			Total any `json:"total"`
		} `json:"hits"`
	}
	require.NoError(t, json.Unmarshal(raw, &root))
	switch total := root.Hits.Total.(type) {
	case float64:
		require.GreaterOrEqual(t, int(total), 1)
	case map[string]any:
		value, ok := total["value"].(float64)
		require.True(t, ok, "hits.total.value missing from response: %s", string(raw))
		require.GreaterOrEqual(t, int(value), 1)
	default:
		t.Fatalf("unsupported hits.total shape in response: %s", string(raw))
	}
}

func envInt(t *testing.T, name string) int {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(name))
	require.NotEmpty(t, raw, "%s must be set", name)
	value, err := strconv.Atoi(raw)
	require.NoError(t, err, "parse %s", name)
	return value
}

func searchBody(major int) json.RawMessage {
	if major <= 5 {
		return json.RawMessage(`{"query":{"term":{"items_count":7}}}`)
	}
	return json.RawMessage(`{"query":{"term":{"items_count":{"value":7}}}}`)
}
