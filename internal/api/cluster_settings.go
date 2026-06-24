package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/elastic"
)

type ClusterSettingsGetIn struct {
	Body HostBody
}

type ClusterSettingsSaveIn struct {
	Body struct {
		HostBody
		Settings json.RawMessage `json:"settings" required:"true" doc:"Cluster settings document to PUT to _cluster/settings."`
	}
}

func (d *Deps) RegisterClusterSettings(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "cluster-settings-get",
		Method:      http.MethodPost,
		Path:        "/cluster_settings",
		Summary:     "Get cluster settings",
		Description: "Returns the cluster settings (persistent and transient) as reported by Elasticsearch.",
		Tags:        []string{"cluster"},
	}, func(ctx context.Context, in *ClusterSettingsGetIn) (*RawOutput, error) {
		return d.passthrough(ctx, in.Body, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.GetClusterSettings(c, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "cluster-settings-save",
		Method:      http.MethodPost,
		Path:        "/cluster_settings/save",
		Summary:     "Save cluster settings",
		Description: "Updates the cluster settings with the given document.",
		Tags:        []string{"cluster"},
	}, func(ctx context.Context, in *ClusterSettingsSaveIn) (*RawOutput, error) {
		return d.passthrough(ctx, in.Body.HostBody, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.SaveClusterSettings(c, in.Body.Settings, t)
		})
	})
}
