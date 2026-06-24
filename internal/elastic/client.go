package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/lmenezes/cerebro/internal/config"
)

type Server struct {
	Host    config.Host
	Headers [][2]string
}

type Response struct {
	Status int
	Body   json.RawMessage
}

func (r Response) IsSuccess() bool {
	return r.Status >= 200 && r.Status < 300
}

type Client interface {
	Main(ctx context.Context, t Server) (Response, error)
	ClusterState(ctx context.Context, t Server) (Response, error)
	IndicesStats(ctx context.Context, t Server) (Response, error)
	NodesStats(ctx context.Context, stats []string, t Server) (Response, error)
	NodeStats(ctx context.Context, node string, t Server) (Response, error)
	IndexStats(ctx context.Context, index string, t Server) (Response, error)
	ClusterSettings(ctx context.Context, t Server) (Response, error)
	Aliases(ctx context.Context, t Server) (Response, error)
	ClusterHealth(ctx context.Context, t Server) (Response, error)
	Nodes(ctx context.Context, flags []string, t Server) (Response, error)
	CloseIndex(ctx context.Context, index string, t Server) (Response, error)
	OpenIndex(ctx context.Context, index string, t Server) (Response, error)
	RefreshIndex(ctx context.Context, index string, t Server) (Response, error)
	FlushIndex(ctx context.Context, index string, t Server) (Response, error)
	ForceMerge(ctx context.Context, index string, t Server) (Response, error)
	ClearIndexCache(ctx context.Context, index string, t Server) (Response, error)
	DeleteIndex(ctx context.Context, index string, t Server) (Response, error)
	GetIndexSettings(ctx context.Context, index string, t Server) (Response, error)
	GetIndexSettingsFlat(ctx context.Context, index string, t Server) (Response, error)
	GetIndexMapping(ctx context.Context, index string, t Server) (Response, error)
	PutClusterSettings(ctx context.Context, settings string, t Server) (Response, error)
	EnableShardAllocation(ctx context.Context, t Server) (Response, error)
	DisableShardAllocation(ctx context.Context, kind string, t Server) (Response, error)
	GetShardStats(ctx context.Context, index string, t Server) (Response, error)
	RelocateShard(ctx context.Context, shard int, index, from, to string, t Server) (Response, error)
	GetIndexRecovery(ctx context.Context, index string, t Server) (Response, error)
	GetClusterMapping(ctx context.Context, t Server) (Response, error)
	GetAliases(ctx context.Context, t Server) (Response, error)
	UpdateAliases(ctx context.Context, changes []json.RawMessage, t Server) (Response, error)
	GetIndexMetadata(ctx context.Context, index string, t Server) (Response, error)
	CreateIndex(ctx context.Context, index string, metadata json.RawMessage, t Server) (Response, error)
	GetIndices(ctx context.Context, t Server) (Response, error)
	GetTemplates(ctx context.Context, t Server) (Response, error)
	CreateTemplate(ctx context.Context, name string, template json.RawMessage, t Server) (Response, error)
	DeleteTemplate(ctx context.Context, name string, t Server) (Response, error)
	GetNodes(ctx context.Context, t Server) (Response, error)
	AnalyzeTextByField(ctx context.Context, index, field, text string, t Server) (Response, error)
	AnalyzeTextByAnalyzer(ctx context.Context, index, analyzer, text string, t Server) (Response, error)
	GetClusterSettings(ctx context.Context, t Server) (Response, error)
	GetRepositories(ctx context.Context, t Server) (Response, error)
	CreateRepository(ctx context.Context, name, repoType string, settings json.RawMessage, t Server) (Response, error)
	DeleteRepository(ctx context.Context, name string, t Server) (Response, error)
	GetSnapshots(ctx context.Context, repo string, t Server) (Response, error)
	DeleteSnapshot(ctx context.Context, repo, snapshot string, t Server) (Response, error)
	CreateSnapshot(ctx context.Context, repo, snapshot string, ignoreUnavailable, includeGlobalState bool, indices *string, t Server) (Response, error)
	RestoreSnapshot(ctx context.Context, repo, snapshot string, renamePattern, renameReplacement *string, ignoreUnavailable, includeAliases, includeGlobalState bool, indices *string, t Server) (Response, error)
	SaveClusterSettings(ctx context.Context, settings json.RawMessage, t Server) (Response, error)
	UpdateIndexSettings(ctx context.Context, index string, settings json.RawMessage, t Server) (Response, error)
	CatRequest(ctx context.Context, api string, t Server) (Response, error)
	CatMaster(ctx context.Context, t Server) (Response, error)
	SearchIndexDocuments(ctx context.Context, index string, query json.RawMessage, t Server) (Response, error)
	SaveIndexDocument(ctx context.Context, index, id string, document json.RawMessage, t Server) (Response, error)
	ExecuteRequest(ctx context.Context, method, path string, data json.RawMessage, t Server) (Response, error)
}

type HTTPClient struct {
	transport        http.RoundTripper
	timeout          time.Duration
	maxResponseBytes int64
}

func NewHTTPClient(hc *http.Client) *HTTPClient {
	timeout := 60 * time.Second
	var transport http.RoundTripper
	if hc == nil {
		hc = http.DefaultClient
	} else {
		timeout = hc.Timeout
	}
	if hc.Transport != nil {
		transport = hc.Transport
	}
	return &HTTPClient{
		transport:        transport,
		timeout:          timeout,
		maxResponseBytes: config.DefaultMaxResponseBytes,
	}
}

func NewHTTPClientWithConfig(hc *http.Client, cfg config.ES) *HTTPClient {
	c := NewHTTPClient(hc)
	if cfg.MaxResponseBytes > 0 {
		c.maxResponseBytes = cfg.MaxResponseBytes
	}
	return c
}

const (
	contentJSON   = "application/json"
	contentNDJSON = "application/x-ndjson"
)

func encoded(s string) string { return url.QueryEscape(s) }

func (c *HTTPClient) execute(ctx context.Context, uri, method string, body []byte, t Server, headers [][2]string) (Response, error) {
	base, err := normalizeBaseURL(t.Host.Host)
	if err != nil {
		return Response{}, err
	}
	if uri == "" {
		uri = "/"
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok && c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	es, err := c.elasticsearchClient(base, t.Host.Auth)
	if err != nil {
		return Response{}, err
	}

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, uri, reader)
	if err != nil {
		return Response{}, err
	}
	for _, h := range headers {
		req.Header.Set(h[0], h[1])
	}
	for _, h := range t.Headers {
		req.Header.Set(h[0], h[1])
	}

	resp, err := es.Perform(req)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()
	raw, err := readLimited(resp.Body, c.maxResponseBytes)
	if err != nil {
		return Response{}, err
	}
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	if !json.Valid(raw) {
		raw, _ = json.Marshal(map[string]string{"error": string(raw)})
	}
	return Response{Status: resp.StatusCode, Body: raw}, nil
}

func (c *HTTPClient) elasticsearchClient(base string, auth *config.ESAuth) (*elasticsearch.Client, error) {
	cfg := elasticsearch.Config{
		Addresses:         []string{base},
		DisableRetry:      true,
		DisableMetaHeader: true,
	}
	if c.transport != nil {
		cfg.Transport = c.transport
	}
	if auth != nil {
		cfg.Username = auth.Username
		cfg.Password = auth.Password
	}
	return elasticsearch.NewClient(cfg)
}

func normalizeBaseURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("invalid elasticsearch host: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("elasticsearch host must use http or https")
	}
	if u.Host == "" {
		return "", errors.New("elasticsearch host must include a host")
	}
	if u.User != nil {
		return "", errors.New("credentials in elasticsearch host URL are not allowed")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func readLimited(r io.Reader, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		return io.ReadAll(r)
	}
	limited := io.LimitReader(r, maxBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > maxBytes {
		return nil, fmt.Errorf("elasticsearch response exceeds configured limit of %d bytes", maxBytes)
	}
	return raw, nil
}

func (c *HTTPClient) Main(ctx context.Context, t Server) (Response, error) {
	return c.execute(ctx, "", http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) ClusterState(ctx context.Context, t Server) (Response, error) {
	return c.execute(ctx, "/_cluster/state/master_node,routing_table,routing_nodes,blocks", http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) IndicesStats(ctx context.Context, t Server) (Response, error) {
	return c.execute(ctx, "/_stats/docs,store", http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) NodesStats(ctx context.Context, stats []string, t Server) (Response, error) {
	p := fmt.Sprintf("/_nodes/stats/%s?human=true", strings.Join(stats, ","))
	return c.execute(ctx, p, http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) NodeStats(ctx context.Context, node string, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/_nodes/%s/stats?human", encoded(node)), http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) IndexStats(ctx context.Context, index string, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/%s/_stats?human=true", encoded(index)), http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) ClusterSettings(ctx context.Context, t Server) (Response, error) {
	return c.execute(ctx, "/_cluster/settings", http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) Aliases(ctx context.Context, t Server) (Response, error) {
	return c.execute(ctx, "/_aliases", http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) ClusterHealth(ctx context.Context, t Server) (Response, error) {
	return c.execute(ctx, "/_cluster/health", http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) Nodes(ctx context.Context, flags []string, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/_nodes/_all/%s?human=true", strings.Join(flags, ",")), http.MethodGet, nil, t, nil)
}

var jsonHeader = [2]string{"Content-type", contentJSON}
var ndjsonHeader = [2]string{"Content-type", contentNDJSON}

func (c *HTTPClient) CloseIndex(ctx context.Context, index string, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/%s/_close", encoded(index)), http.MethodPost, nil, t, [][2]string{jsonHeader})
}

func (c *HTTPClient) OpenIndex(ctx context.Context, index string, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/%s/_open", encoded(index)), http.MethodPost, nil, t, [][2]string{jsonHeader})
}

func (c *HTTPClient) RefreshIndex(ctx context.Context, index string, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/%s/_refresh", encoded(index)), http.MethodPost, nil, t, [][2]string{jsonHeader})
}

func (c *HTTPClient) FlushIndex(ctx context.Context, index string, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/%s/_flush", encoded(index)), http.MethodPost, nil, t, [][2]string{jsonHeader})
}

func (c *HTTPClient) ForceMerge(ctx context.Context, index string, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/%s/_forcemerge", encoded(index)), http.MethodPost, nil, t, [][2]string{jsonHeader})
}

func (c *HTTPClient) ClearIndexCache(ctx context.Context, index string, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/%s/_cache/clear", encoded(index)), http.MethodPost, nil, t, [][2]string{jsonHeader})
}

func (c *HTTPClient) DeleteIndex(ctx context.Context, index string, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/%s", encoded(index)), http.MethodDelete, nil, t, nil)
}

func (c *HTTPClient) GetIndexSettings(ctx context.Context, index string, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/%s/_settings", encoded(index)), http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) GetIndexSettingsFlat(ctx context.Context, index string, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/%s/_settings?flat_settings=true&include_defaults=true", encoded(index)), http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) GetIndexMapping(ctx context.Context, index string, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/%s/_mapping", encoded(index)), http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) PutClusterSettings(ctx context.Context, settings string, t Server) (Response, error) {
	return c.execute(ctx, "/_cluster/settings", http.MethodPut, []byte(settings), t, [][2]string{jsonHeader})
}

func allocationSettings(value string) string {
	return fmt.Sprintf(`{"transient": {"cluster": {"routing": {"allocation": {"enable": "%s"}}}}}`, value)
}

func (c *HTTPClient) EnableShardAllocation(ctx context.Context, t Server) (Response, error) {
	return c.PutClusterSettings(ctx, allocationSettings("all"), t)
}

func (c *HTTPClient) DisableShardAllocation(ctx context.Context, kind string, t Server) (Response, error) {
	return c.PutClusterSettings(ctx, allocationSettings(kind), t)
}

func (c *HTTPClient) GetShardStats(ctx context.Context, index string, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/%s/_stats?level=shards&human=true", encoded(index)), http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) RelocateShard(ctx context.Context, shard int, index, from, to string, t Server) (Response, error) {
	body, _ := json.Marshal(map[string]any{
		"commands": []any{
			map[string]any{
				"move": map[string]any{
					"shard":     shard,
					"index":     index,
					"from_node": from,
					"to_node":   to,
				},
			},
		},
	})
	return c.execute(ctx, "/_cluster/reroute", http.MethodPost, body, t, [][2]string{jsonHeader})
}

func (c *HTTPClient) GetIndexRecovery(ctx context.Context, index string, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/%s/_recovery?active_only=true&human=true", encoded(index)), http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) GetClusterMapping(ctx context.Context, t Server) (Response, error) {
	return c.execute(ctx, "/_mapping", http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) GetAliases(ctx context.Context, t Server) (Response, error) {
	return c.execute(ctx, "/_aliases", http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) UpdateAliases(ctx context.Context, changes []json.RawMessage, t Server) (Response, error) {
	body := map[string][]json.RawMessage{"actions": changes}
	if changes == nil {
		body["actions"] = []json.RawMessage{}
	}
	raw, _ := json.Marshal(body)
	return c.execute(ctx, "/_aliases", http.MethodPost, raw, t, [][2]string{jsonHeader})
}

func (c *HTTPClient) GetIndexMetadata(ctx context.Context, index string, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/_cluster/state/metadata/%s?human=true", encoded(index)), http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) CreateIndex(ctx context.Context, index string, metadata json.RawMessage, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/%s", encoded(index)), http.MethodPut, metadata, t, [][2]string{jsonHeader})
}

func (c *HTTPClient) GetIndices(ctx context.Context, t Server) (Response, error) {
	return c.execute(ctx, "/_cat/indices?format=json", http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) GetTemplates(ctx context.Context, t Server) (Response, error) {
	return c.execute(ctx, "/_template", http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) CreateTemplate(ctx context.Context, name string, template json.RawMessage, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/_template/%s", encoded(name)), http.MethodPut, template, t, [][2]string{jsonHeader})
}

func (c *HTTPClient) DeleteTemplate(ctx context.Context, name string, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/_template/%s", encoded(name)), http.MethodDelete, nil, t, nil)
}

func (c *HTTPClient) GetNodes(ctx context.Context, t Server) (Response, error) {
	return c.execute(ctx, "/_cat/nodes?format=json", http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) AnalyzeTextByField(ctx context.Context, index, field, text string, t Server) (Response, error) {
	body, _ := json.Marshal(map[string]string{"text": text, "field": field})
	return c.execute(ctx, fmt.Sprintf("/%s/_analyze", encoded(index)), http.MethodGet, body, t, [][2]string{jsonHeader})
}

func (c *HTTPClient) AnalyzeTextByAnalyzer(ctx context.Context, index, analyzer, text string, t Server) (Response, error) {
	body, _ := json.Marshal(map[string]string{"text": text, "analyzer": analyzer})
	return c.execute(ctx, fmt.Sprintf("/%s/_analyze", encoded(index)), http.MethodGet, body, t, [][2]string{jsonHeader})
}

func (c *HTTPClient) GetClusterSettings(ctx context.Context, t Server) (Response, error) {
	return c.execute(ctx, "/_cluster/settings?flat_settings=true&include_defaults=true", http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) GetRepositories(ctx context.Context, t Server) (Response, error) {
	return c.execute(ctx, "/_snapshot", http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) CreateRepository(ctx context.Context, name, repoType string, settings json.RawMessage, t Server) (Response, error) {
	body := map[string]any{"type": repoType, "settings": settings}
	raw, _ := json.Marshal(body)
	return c.execute(ctx, fmt.Sprintf("/_snapshot/%s", encoded(name)), http.MethodPut, raw, t, [][2]string{jsonHeader})
}

func (c *HTTPClient) DeleteRepository(ctx context.Context, name string, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/_snapshot/%s", encoded(name)), http.MethodDelete, nil, t, nil)
}

func (c *HTTPClient) GetSnapshots(ctx context.Context, repo string, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/_snapshot/%s/_all", encoded(repo)), http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) DeleteSnapshot(ctx context.Context, repo, snapshot string, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/_snapshot/%s/%s", encoded(repo), encoded(snapshot)), http.MethodDelete, nil, t, nil)
}

func (c *HTTPClient) CreateSnapshot(ctx context.Context, repo, snapshot string, ignoreUnavailable, includeGlobalState bool, indices *string, t Server) (Response, error) {
	body := map[string]any{
		"repository":         repo,
		"snapshot":           snapshot,
		"ignoreUnavailable":  ignoreUnavailable,
		"includeGlobalState": includeGlobalState,
	}
	if indices != nil {
		body["indices"] = *indices
	}
	raw, _ := json.Marshal(body)
	return c.execute(ctx, fmt.Sprintf("/_snapshot/%s/%s", encoded(repo), encoded(snapshot)), http.MethodPut, raw, t, [][2]string{jsonHeader})
}

func (c *HTTPClient) RestoreSnapshot(ctx context.Context, repo, snapshot string, renamePattern, renameReplacement *string, ignoreUnavailable, includeAliases, includeGlobalState bool, indices *string, t Server) (Response, error) {
	body := map[string]any{
		"ignore_unavailable":   ignoreUnavailable,
		"include_global_state": includeGlobalState,
		"include_aliases":      includeAliases,
	}
	if indices != nil {
		body["indices"] = *indices
	}
	if renamePattern != nil {
		body["rename_pattern"] = *renamePattern
	}
	if renameReplacement != nil {
		body["rename_replacement"] = *renameReplacement
	}
	raw, _ := json.Marshal(body)
	return c.execute(ctx, fmt.Sprintf("/_snapshot/%s/%s/_restore", encoded(repo), encoded(snapshot)), http.MethodPost, raw, t, [][2]string{jsonHeader})
}

func (c *HTTPClient) SaveClusterSettings(ctx context.Context, settings json.RawMessage, t Server) (Response, error) {
	return c.execute(ctx, "/_cluster/settings", http.MethodPut, settings, t, [][2]string{jsonHeader})
}

func (c *HTTPClient) UpdateIndexSettings(ctx context.Context, index string, settings json.RawMessage, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/%s/_settings", encoded(index)), http.MethodPut, settings, t, [][2]string{jsonHeader})
}

func (c *HTTPClient) CatRequest(ctx context.Context, api string, t Server) (Response, error) {
	path, err := catPath(api)
	if err != nil {
		return Response{}, err
	}
	return c.execute(ctx, path, http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) CatMaster(ctx context.Context, t Server) (Response, error) {
	return c.execute(ctx, "/_cat/master?format=json", http.MethodGet, nil, t, nil)
}

func (c *HTTPClient) SearchIndexDocuments(ctx context.Context, index string, query json.RawMessage, t Server) (Response, error) {
	return c.execute(ctx, fmt.Sprintf("/%s/_search", encoded(index)), http.MethodPost, query, t, [][2]string{jsonHeader})
}

func (c *HTTPClient) SaveIndexDocument(ctx context.Context, index, id string, document json.RawMessage, t Server) (Response, error) {
	if strings.TrimSpace(id) == "" {
		return c.execute(ctx, fmt.Sprintf("/%s/_doc?refresh=true", encoded(index)), http.MethodPost, document, t, [][2]string{jsonHeader})
	}
	return c.execute(ctx, fmt.Sprintf("/%s/_doc/%s?refresh=true", encoded(index), encoded(id)), http.MethodPut, document, t, [][2]string{jsonHeader})
}

func (c *HTTPClient) ExecuteRequest(ctx context.Context, method, path string, data json.RawMessage, t Server) (Response, error) {
	uri, err := restPath(path)
	if err != nil {
		return Response{}, err
	}
	var body []byte
	var headers [][2]string
	if data != nil {
		// if data is a JSON string, treat it as raw content (bulk/multisearch ndjson)
		var asString string
		if err := json.Unmarshal(data, &asString); err == nil {
			body = []byte(asString)
			headers = [][2]string{ndjsonHeader}
		} else {
			body = data
			headers = [][2]string{jsonHeader}
		}
	}
	return c.execute(ctx, uri, method, body, t, headers)
}

func catPath(api string) (string, error) {
	switch strings.TrimSpace(api) {
	case "aliases":
		return "/_cat/aliases?format=json", nil
	case "allocation":
		return "/_cat/allocation?format=json", nil
	case "count":
		return "/_cat/count?format=json", nil
	case "fielddata":
		return "/_cat/fielddata?format=json", nil
	case "health":
		return "/_cat/health?format=json", nil
	case "indices":
		return "/_cat/indices?format=json", nil
	case "master":
		return "/_cat/master?format=json", nil
	case "nodes":
		return "/_cat/nodes?format=json", nil
	case "pending tasks", "pending_tasks":
		return "/_cat/pending_tasks?format=json", nil
	case "plugins":
		return "/_cat/plugins?format=json", nil
	case "recovery":
		return "/_cat/recovery?format=json", nil
	case "repositories":
		return "/_cat/repositories?format=json", nil
	case "shards":
		return "/_cat/shards?format=json", nil
	case "segments":
		return "/_cat/segments?format=json", nil
	case "thread pool", "thread_pool":
		return "/_cat/thread_pool?format=json", nil
	default:
		return "", fmt.Errorf("unsupported cat API: %s", api)
	}
}

func restPath(path string) (string, error) {
	p := strings.TrimSpace(path)
	if p == "" {
		return "", errors.New("missing request path")
	}
	if strings.ContainsAny(p, "\x00\r\n") {
		return "", errors.New("request path contains invalid characters")
	}
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") || strings.HasPrefix(p, "//") {
		return "", errors.New("request path must be relative to the selected Elasticsearch host")
	}
	for strings.HasPrefix(p, "/") {
		p = strings.TrimPrefix(p, "/")
	}
	if p == "" {
		return "/", nil
	}
	return "/" + p, nil
}
