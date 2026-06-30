package api

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/elastic"
	"github.com/lmenezes/cerebro/internal/transform"
)

const (
	dataExplorerDefaultSize = 25
	dataExplorerMaxIDLen    = 512
	dataExplorerMaxPage     = 10000
	dataExplorerMaxQueryLen = 512
	dataExplorerMaxSize     = 100
)

var dataExplorerSortField = regexp.MustCompile(`^[_@A-Za-z0-9][_.@A-Za-z0-9-]{0,127}$`)
var dataExplorerFieldSpacing = regexp.MustCompile(`([_@A-Za-z0-9][_.@A-Za-z0-9-]{0,127}):\s+("[^"]+"|\S+)`)
var dataExplorerRangeOperator = regexp.MustCompile(`([_@A-Za-z0-9][_.@A-Za-z0-9-]{0,127})\s*(>=|>|<=|<)\s*("[^"]+"|\S+)`)
var dataExplorerBoolOperator = regexp.MustCompile(`(?i)\b(and|or|not)\b`)

type DataExplorerSearchIn struct {
	ClusterPath
	Index string `path:"index" doc:"Index name."`
	Body  struct {
		Page      int    `json:"page,omitempty" minimum:"0" maximum:"10000" doc:"Zero-based page number."`
		Size      int    `json:"size,omitempty" minimum:"1" maximum:"100" doc:"Rows per page."`
		Query     string `json:"query,omitempty" maxLength:"512" doc:"Optional simple query string."`
		QueryMode string `json:"query_mode,omitempty" enum:"kql,lucene" doc:"Query language mode. KQL is the Kibana-like default; Lucene maps to Elasticsearch query_string syntax."`
		SortField string `json:"sort_field,omitempty" doc:"Optional field to sort by."`
		SortOrder string `json:"sort_order,omitempty" enum:"asc,desc" doc:"Sort direction."`
	}
}

type DataExplorerSaveIn struct {
	ClusterPath
	Index string `path:"index" doc:"Index name."`
	Body  struct {
		ID       string          `json:"id,omitempty" maxLength:"512" doc:"Optional document id. Empty creates a new document id."`
		Document json.RawMessage `json:"document" required:"true" doc:"Document _source JSON object."`
	}
}

func (d *Deps) RegisterDataExplorer(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "data-explorer-search",
		Method:      http.MethodPost,
		Path:        "/data_explorer/{index}/search",
		Summary:     "Browse index documents",
		Description: "Read-only, feature-flagged document browser for one Elasticsearch index.",
		Tags:        []string{"data-explorer"},
	}, func(ctx context.Context, in *DataExplorerSearchIn) (*Output[transform.DataExplorerResult], error) {
		if !d.Cfg.Features.DataExplorer {
			return failMsg[transform.DataExplorerResult](http.StatusNotFound, "data explorer is disabled")
		}
		if err := validateDataExplorerRequest(in); err != nil {
			return failMsg[transform.DataExplorerResult](http.StatusBadRequest, err.Error())
		}
		query, err := dataExplorerQuery(in)
		if err != nil {
			return failMsg[transform.DataExplorerResult](http.StatusBadRequest, err.Error())
		}
		return transformResp(ctx, d,
			func(c context.Context, t elastic.Server) (elastic.Response, error) {
				return d.Client.SearchIndexDocuments(c, in.Index, query, t)
			},
			transform.DataExplorerSearch)
	})

	huma.Register(api, huma.Operation{
		OperationID: "data-explorer-save",
		Method:      http.MethodPut,
		Path:        "/data_explorer/{index}/documents",
		Summary:     "Save index document",
		Description: "Feature-flagged document insert/update for one Elasticsearch index.",
		Tags:        []string{"data-explorer"},
	}, func(ctx context.Context, in *DataExplorerSaveIn) (*RawOutput, error) {
		if !d.Cfg.Features.DataExplorer {
			return failMsg[RawResponse](http.StatusNotFound, "data explorer is disabled")
		}
		if err := validateDataExplorerSave(in); err != nil {
			return failMsg[RawResponse](http.StatusBadRequest, err.Error())
		}
		t, err := clusterTarget(ctx)
		if err != nil {
			return failMsg[RawResponse](http.StatusBadRequest, err.Error())
		}
		settings, err := d.Client.GetIndexSettings(ctx, in.Index, t)
		if err != nil {
			return failMsg[RawResponse](http.StatusInternalServerError, err.Error())
		}
		if !settings.IsSuccess() {
			return fail[RawResponse](settings.Status, settings.Body)
		}
		if dataExplorerIndexReadOnly(settings.Body, in.Index) {
			return failMsg[RawResponse](http.StatusConflict, "index is read-only")
		}
		resp, err := d.Client.SaveIndexDocument(ctx, in.Index, in.Body.ID, in.Body.Document, t)
		if err != nil {
			return failMsg[RawResponse](http.StatusInternalServerError, err.Error())
		}
		return raw(resp.Status, resp.Body)
	})
}

func validateDataExplorerRequest(in *DataExplorerSearchIn) error {
	index := strings.TrimSpace(in.Index)
	if err := validateDataExplorerIndex(index); err != nil {
		return err
	}
	in.Index = index
	if len(in.Body.Query) > dataExplorerMaxQueryLen {
		return huma.Error400BadRequest("query is too long")
	}
	in.Body.QueryMode = strings.ToLower(strings.TrimSpace(in.Body.QueryMode))
	if in.Body.QueryMode == "" {
		in.Body.QueryMode = "kql"
	}
	if in.Body.QueryMode != "kql" && in.Body.QueryMode != "lucene" {
		return huma.Error400BadRequest("query mode must be kql or lucene")
	}
	if in.Body.Page < 0 || in.Body.Page > dataExplorerMaxPage {
		return huma.Error400BadRequest("page is out of range")
	}
	size := in.Body.Size
	if size == 0 {
		size = dataExplorerDefaultSize
	}
	if size < 1 || size > dataExplorerMaxSize {
		return huma.Error400BadRequest("size is out of range")
	}
	sortField := strings.TrimSpace(in.Body.SortField)
	if sortField != "" && sortField != "_score" && sortField != "_doc" && !dataExplorerSortField.MatchString(sortField) {
		return huma.Error400BadRequest("sort field contains unsupported characters")
	}
	sortOrder := strings.ToLower(strings.TrimSpace(in.Body.SortOrder))
	if sortOrder != "" && sortOrder != "asc" && sortOrder != "desc" {
		return huma.Error400BadRequest("sort order must be asc or desc")
	}
	return nil
}

func validateDataExplorerSave(in *DataExplorerSaveIn) error {
	index := strings.TrimSpace(in.Index)
	if err := validateDataExplorerIndex(index); err != nil {
		return err
	}
	in.Index = index
	in.Body.ID = strings.TrimSpace(in.Body.ID)
	if len(in.Body.ID) > dataExplorerMaxIDLen || strings.ContainsAny(in.Body.ID, "\x00\r\n") {
		return huma.Error400BadRequest("document id contains unsupported characters")
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(in.Body.Document, &document); err != nil || document == nil {
		return huma.Error400BadRequest("document must be a JSON object")
	}
	return nil
}

func validateDataExplorerIndex(index string) error {
	if index == "" {
		return huma.Error400BadRequest("index is required")
	}
	if strings.ContainsAny(index, "\x00\r\n/*?,\\") {
		return huma.Error400BadRequest("index must be a concrete index name")
	}
	return nil
}

func dataExplorerQuery(in *DataExplorerSearchIn) (json.RawMessage, error) {
	size := in.Body.Size
	if size == 0 {
		size = dataExplorerDefaultSize
	}
	body := map[string]any{
		"from":             in.Body.Page * size,
		"size":             size,
		"track_total_hits": true,
	}
	if query := dataExplorerSearchQuery(in.Body.Query, in.Body.QueryMode); query != "" {
		body["query"] = map[string]any{
			"query_string": map[string]any{
				"analyze_wildcard": true,
				"query":            query,
				"default_operator": "AND",
				"lenient":          true,
			},
		}
	} else {
		body["query"] = map[string]any{"match_all": map[string]any{}}
	}
	sortField := strings.TrimSpace(in.Body.SortField)
	sortOrder := strings.ToLower(strings.TrimSpace(in.Body.SortOrder))
	if sortOrder == "" {
		sortOrder = "asc"
	}
	if sortField != "" {
		switch sortField {
		case "_doc":
			body["sort"] = []any{"_doc"}
		case "_score":
			body["sort"] = []any{map[string]any{"_score": map[string]any{"order": sortOrder}}}
		default:
			body["sort"] = []any{map[string]any{sortField: map[string]any{"order": sortOrder, "unmapped_type": "keyword"}}}
		}
	} else {
		body["sort"] = []any{"_doc"}
	}
	return json.Marshal(body)
}

func normalizeDataExplorerQuery(query string) string {
	return dataExplorerFieldSpacing.ReplaceAllString(strings.TrimSpace(query), "$1:$2")
}

func dataExplorerSearchQuery(query, mode string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}
	if mode == "lucene" {
		return normalizeDataExplorerQuery(query)
	}
	return normalizeKQLQuery(query)
}

func normalizeKQLQuery(query string) string {
	query = normalizeDataExplorerQuery(query)
	query = dataExplorerRangeOperator.ReplaceAllString(query, "$1:$2$3")
	query = dataExplorerBoolOperator.ReplaceAllStringFunc(query, strings.ToUpper)
	return query
}

func dataExplorerIndexReadOnly(raw json.RawMessage, index string) bool {
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return false
	}
	target, _ := root[index].(map[string]any)
	if target == nil {
		for _, value := range root {
			if candidate, ok := value.(map[string]any); ok {
				target = candidate
				break
			}
		}
	}
	if target == nil {
		return false
	}
	return truthySetting(getNestedAny(target, "settings", "index", "blocks", "write")) ||
		truthySetting(getNestedAny(target, "settings", "index", "blocks", "read_only")) ||
		truthySetting(getNestedAny(target, "settings", "index", "blocks", "read_only_allow_delete")) ||
		truthySetting(getNestedAny(target, "settings", "index.blocks.write")) ||
		truthySetting(getNestedAny(target, "settings", "index.blocks.read_only")) ||
		truthySetting(getNestedAny(target, "settings", "index.blocks.read_only_allow_delete"))
}

func truthySetting(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true")
	default:
		return false
	}
}

func getNestedAny(value any, path ...string) any {
	current := value
	for _, key := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[key]
	}
	return current
}
