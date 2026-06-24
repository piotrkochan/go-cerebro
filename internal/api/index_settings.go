package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/elastic"
)

type IndexSettingsGetIn struct {
	Body struct {
		HostBody
		Index string `json:"index" required:"true" doc:"Index name."`
	}
}

type IndexSettingsUpdateIn struct {
	Body struct {
		HostBody
		Index    string          `json:"index" required:"true" doc:"Index name."`
		Settings json.RawMessage `json:"settings" required:"true" doc:"Settings document to PUT to the index _settings."`
	}
}

func (d *Deps) RegisterIndexSettings(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "index-settings-get",
		Method:      http.MethodPost,
		Path:        "/index_settings",
		Summary:     "Get index settings (flat)",
		Description: "Returns the index settings in flat format, including defaults.",
		Tags:        []string{"index"},
	}, func(ctx context.Context, in *IndexSettingsGetIn) (*RawOutput, error) {
		return d.passthrough(ctx, in.Body.HostBody, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.GetIndexSettingsFlat(c, in.Body.Index, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "index-settings-update",
		Method:      http.MethodPost,
		Path:        "/index_settings/update",
		Summary:     "Update index settings",
		Tags:        []string{"index"},
	}, func(ctx context.Context, in *IndexSettingsUpdateIn) (*RawOutput, error) {
		return d.passthrough(ctx, in.Body.HostBody, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.UpdateIndexSettings(c, in.Body.Index, in.Body.Settings, t)
		})
	})
}
