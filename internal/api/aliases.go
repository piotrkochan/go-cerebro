package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/elastic"
	"github.com/lmenezes/cerebro/internal/transform"
)

type AliasesGetIn struct {
	ClusterPath
}

type AliasesUpdateIn struct {
	ClusterPath
	Body struct {
		Changes []json.RawMessage `json:"changes,omitempty" doc:"Alias actions in Elasticsearch _aliases format, e.g. {\"add\": {...}} / {\"remove\": {...}}."`
	}
}

func (d *Deps) RegisterAliases(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "aliases-get",
		Method:      http.MethodGet,
		Path:        "/aliases",
		Summary:     "List aliases",
		Description: "Returns all index aliases flattened into one list.",
		Tags:        []string{"aliases"},
	}, func(ctx context.Context, in *AliasesGetIn) (*Output[List[transform.Alias]], error) {
		return transformListResp(ctx, d,
			func(c context.Context, t elastic.Server) (elastic.Response, error) { return d.Client.GetAliases(c, t) },
			transform.Aliases)
	})

	huma.Register(api, huma.Operation{
		OperationID: "aliases-update",
		Method:      http.MethodPut,
		Path:        "/aliases",
		Summary:     "Update aliases",
		Description: "Applies a batch of alias add/remove actions.",
		Tags:        []string{"aliases"},
	}, func(ctx context.Context, in *AliasesUpdateIn) (*RawOutput, error) {
		t, err := clusterTarget(ctx)
		if err != nil {
			return failMsg[RawResponse](400, err.Error())
		}
		resp, err := d.Client.UpdateAliases(ctx, in.Body.Changes, t)
		if err != nil {
			return failMsg[RawResponse](500, err.Error())
		}
		return raw(resp.Status, resp.Body)
	})
}
