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

type ConnectHostsResponse struct {
	Items           []config.HostRef `json:"items" doc:"Configured Elasticsearch hosts."`
	AllowAdHocHosts bool             `json:"allow_ad_hoc_hosts" doc:"Whether users may enter an arbitrary Elasticsearch URL."`
}

func (d *Deps) RegisterConnect(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "connect-hosts",
		Method:      http.MethodGet,
		Path:        "/connect/hosts",
		Summary:     "List configured hosts",
		Description: "Returns the names of the Elasticsearch hosts configured in Cerebro.",
		Tags:        []string{"connect"},
	}, func(ctx context.Context, _ *struct{}) (*Output[ConnectHostsResponse], error) {
		refs := d.Cfg.HostRefs()
		if refs == nil {
			refs = []config.HostRef{}
		}
		return ok(200, ConnectHostsResponse{Items: refs, AllowAdHocHosts: d.Cfg.ES.AllowAdHocHosts})
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
