package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/elastic"
)

type ConnectIn struct {
	Body HostBody
}

func (d *Deps) RegisterConnect(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "connect-hosts",
		Method:      http.MethodGet,
		Path:        "/connect/hosts",
		Summary:     "List configured hosts",
		Description: "Returns the names of the Elasticsearch hosts configured in Cerebro.",
		Tags:        []string{"connect"},
	}, func(ctx context.Context, _ *struct{}) (*Output[List[string]], error) {
		return okList(200, d.Cfg.HostNames())
	})

	huma.Register(api, huma.Operation{
		OperationID: "connect",
		Method:      http.MethodPost,
		Path:        "/connect",
		Summary:     "Connect to a cluster",
		Description: "Checks connectivity by fetching _cluster/health of the target host.",
		Tags:        []string{"connect"},
	}, func(ctx context.Context, in *ConnectIn) (*RawOutput, error) {
		return d.passthrough(ctx, in.Body, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.ExecuteRequest(c, http.MethodGet, "_cluster/health", nil, t)
		})
	})
}
