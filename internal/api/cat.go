package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type CatIn struct {
	Body struct {
		HostBody
		API string `json:"api" required:"true" doc:"Name of the _cat API to call, e.g. \"indices\", \"shards\", \"allocation\"."`
	}
}

func (d *Deps) RegisterCat(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "cat",
		Method:      http.MethodPost,
		Path:        "/cat",
		Summary:     "Run a _cat API",
		Description: "Proxies the given Elasticsearch _cat API and returns its JSON output verbatim.",
		Tags:        []string{"cat"},
	}, func(ctx context.Context, in *CatIn) (*RawOutput, error) {
		t, err := d.resolveTarget(httpRequest(ctx), in.Body.HostBody)
		if err != nil {
			return failMsg[RawResponse](400, err.Error())
		}
		resp, err := d.Client.CatRequest(ctx, in.Body.API, t)
		if err != nil {
			return failMsg[RawResponse](500, err.Error())
		}
		return raw(resp.Status, resp.Body)
	})
}
