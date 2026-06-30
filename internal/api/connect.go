package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/config"
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
	}, func(ctx context.Context, _ *struct{}) (*Output[List[config.HostRef]], error) {
		return okList(200, d.Cfg.HostRefs())
	})

	huma.Register(api, huma.Operation{
		OperationID: "connect",
		Method:      http.MethodPost,
		Path:        "/connect",
		Summary:     "Connect to a cluster",
		Description: "Checks connectivity by fetching _cluster/health of the target host.",
		Tags:        []string{"connect"},
	}, func(ctx context.Context, in *ConnectIn) (*RawOutput, error) {
		t, err := d.resolveTarget(httpRequest(ctx), in.Body)
		if err != nil {
			return failMsg[RawResponse](400, err.Error())
		}
		resp, err := d.Client.ExecuteRequest(ctx, http.MethodGet, "_cluster/health", nil, t)
		if err != nil {
			return failMsg[RawResponse](500, err.Error())
		}
		return raw(resp.Status, resp.Body)
	})
}
