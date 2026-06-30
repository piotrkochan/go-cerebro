package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type CatIn struct {
	ClusterPath
	API string `path:"api" doc:"Name of the _cat API to call, e.g. \"indices\", \"shards\", \"allocation\"."`
}

func (d *Deps) RegisterCat(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "cat",
		Method:      http.MethodGet,
		Path:        "/cat/{api}",
		Summary:     "Run a _cat API",
		Description: "Proxies the given Elasticsearch _cat API and returns its JSON output verbatim.",
		Tags:        []string{"cat"},
	}, func(ctx context.Context, in *CatIn) (*RawOutput, error) {
		t, err := clusterTarget(ctx)
		if err != nil {
			return failMsg[RawResponse](400, err.Error())
		}
		resp, err := d.Client.CatRequest(ctx, in.API, t)
		if err != nil {
			return failMsg[RawResponse](500, err.Error())
		}
		return raw(resp.Status, resp.Body)
	})
}
