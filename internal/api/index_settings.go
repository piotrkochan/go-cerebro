package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/elastic"
)

type IndexSettingsGetIn struct {
	ClusterPath
	Index string `path:"index" doc:"Index name."`
}

type IndexSettingsUpdateIn struct {
	ClusterPath
	Index string          `path:"index" doc:"Index name."`
	Body  json.RawMessage `required:"true" doc:"Settings document to PUT to the index _settings."`
}

func (d *Deps) RegisterIndexSettings(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "index-settings-get",
		Method:      http.MethodGet,
		Path:        "/index_settings/{index}",
		Summary:     "Get index settings (flat)",
		Description: "Returns the index settings in flat format, including defaults.",
		Tags:        []string{"index"},
	}, func(ctx context.Context, in *IndexSettingsGetIn) (*RawOutput, error) {
		return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.GetIndexSettingsFlat(c, in.Index, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "index-settings-update",
		Method:      http.MethodPut,
		Path:        "/index_settings/{index}",
		Summary:     "Update index settings",
		Tags:        []string{"index"},
	}, func(ctx context.Context, in *IndexSettingsUpdateIn) (*RawOutput, error) {
		return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.UpdateIndexSettings(c, in.Index, in.Body, t)
		})
	})
}
