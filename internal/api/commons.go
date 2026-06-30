package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/elastic"
	"github.com/lmenezes/cerebro/internal/transform"
)

type CommonsHostIn struct {
	ClusterPath
}

type CommonsIndexIn struct {
	ClusterPath
	Index string `path:"index" doc:"Index name."`
}

type CommonsNodeIn struct {
	ClusterPath
	Node string `path:"node" doc:"Node name or id."`
}

func (d *Deps) RegisterCommons(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "commons-indices",
		Method:      http.MethodGet,
		Path:        "/commons/indices",
		Summary:     "List index names",
		Tags:        []string{"commons"},
	}, func(ctx context.Context, in *CommonsHostIn) (*Output[List[string]], error) {
		return transformListResp(ctx, d,
			func(c context.Context, t elastic.Server) (elastic.Response, error) { return d.Client.GetIndices(c, t) },
			transform.CommonsIndices)
	})

	huma.Register(api, huma.Operation{
		OperationID: "commons-nodes",
		Method:      http.MethodGet,
		Path:        "/commons/nodes",
		Summary:     "List node names",
		Tags:        []string{"commons"},
	}, func(ctx context.Context, in *CommonsHostIn) (*Output[List[string]], error) {
		return transformListResp(ctx, d,
			func(c context.Context, t elastic.Server) (elastic.Response, error) { return d.Client.GetNodes(c, t) },
			transform.CommonsNodes)
	})

	huma.Register(api, huma.Operation{
		OperationID: "commons-get-index-settings",
		Method:      http.MethodGet,
		Path:        "/commons/indices/{index}/settings",
		Summary:     "Get index settings",
		Tags:        []string{"commons"},
	}, func(ctx context.Context, in *CommonsIndexIn) (*RawOutput, error) {
		return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.GetIndexSettings(c, in.Index, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "commons-get-index-mapping",
		Method:      http.MethodGet,
		Path:        "/commons/indices/{index}/mapping",
		Summary:     "Get index mapping",
		Tags:        []string{"commons"},
	}, func(ctx context.Context, in *CommonsIndexIn) (*RawOutput, error) {
		return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.GetIndexMapping(c, in.Index, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "commons-get-node-stats",
		Method:      http.MethodGet,
		Path:        "/commons/nodes/{node}/stats",
		Summary:     "Get node stats",
		Tags:        []string{"commons"},
	}, func(ctx context.Context, in *CommonsNodeIn) (*RawOutput, error) {
		return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.NodeStats(c, in.Node, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "commons-get-index-stats",
		Method:      http.MethodGet,
		Path:        "/commons/indices/{index}/stats",
		Summary:     "Get index stats",
		Tags:        []string{"commons"},
	}, func(ctx context.Context, in *CommonsIndexIn) (*RawOutput, error) {
		return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.IndexStats(c, in.Index, t)
		})
	})
}
