package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/elastic"
	"github.com/lmenezes/cerebro/internal/transform"
)

type CreateIndexGetMetaIn struct {
	ClusterPath
	Index string `path:"index" doc:"Index to copy mappings/settings from."`
}

type CreateIndexCreateIn struct {
	ClusterPath
	Index string          `path:"index" doc:"Name of the index to create."`
	Body  json.RawMessage `required:"false" doc:"Index creation body (settings, mappings, aliases). Defaults to {}."`
}

func (d *Deps) RegisterCreateIndex(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "create-index-get-metadata",
		Method:      http.MethodGet,
		Path:        "/create_index/{index}/metadata",
		Summary:     "Get index metadata",
		Description: "Returns the mappings and settings of an existing index, used to prefill the create-index form.",
		Tags:        []string{"create_index"},
	}, func(ctx context.Context, in *CreateIndexGetMetaIn) (*Output[transform.Metadata], error) {
		return transformResp(ctx, d, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.GetIndexMetadata(c, in.Index, t)
		}, transform.IndexMetadata)
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-index-create",
		Method:      http.MethodPut,
		Path:        "/create_index/{index}",
		Summary:     "Create an index",
		Tags:        []string{"create_index"},
	}, func(ctx context.Context, in *CreateIndexCreateIn) (*RawOutput, error) {
		md := in.Body
		if len(md) == 0 {
			md = json.RawMessage(`{}`)
		}
		return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.CreateIndex(c, in.Index, md, t)
		})
	})
}
