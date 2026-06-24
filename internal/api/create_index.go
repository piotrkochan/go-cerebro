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
	Body struct {
		HostBody
		Index string `json:"index" required:"true" doc:"Index to copy mappings/settings from."`
	}
}

type CreateIndexCreateIn struct {
	Body struct {
		HostBody
		Index    string          `json:"index" required:"true" doc:"Name of the index to create."`
		Metadata json.RawMessage `json:"metadata,omitempty" doc:"Index creation body (settings, mappings, aliases). Defaults to {}."`
	}
}

func (d *Deps) RegisterCreateIndex(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "create-index-get-metadata",
		Method:      http.MethodPost,
		Path:        "/create_index/get_index_metadata",
		Summary:     "Get index metadata",
		Description: "Returns the mappings and settings of an existing index, used to prefill the create-index form.",
		Tags:        []string{"create_index"},
	}, func(ctx context.Context, in *CreateIndexGetMetaIn) (*Output[transform.Metadata], error) {
		return transformResp(ctx, d, in.Body.HostBody, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.GetIndexMetadata(c, in.Body.Index, t)
		}, transform.IndexMetadata)
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-index-create",
		Method:      http.MethodPost,
		Path:        "/create_index/create",
		Summary:     "Create an index",
		Tags:        []string{"create_index"},
	}, func(ctx context.Context, in *CreateIndexCreateIn) (*RawOutput, error) {
		md := in.Body.Metadata
		if len(md) == 0 {
			md = json.RawMessage(`{}`)
		}
		return d.passthrough(ctx, in.Body.HostBody, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.CreateIndex(c, in.Body.Index, md, t)
		})
	})
}
