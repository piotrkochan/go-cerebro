package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/elastic"
	"github.com/lmenezes/cerebro/internal/transform"
)

type CommonsHostIn struct {
	Body HostBody
}

type CommonsIndexIn struct {
	Body struct {
		HostBody
		Index string `json:"index" required:"true" doc:"Index name."`
	}
}

type CommonsNodeIn struct {
	Body struct {
		HostBody
		Node string `json:"node" required:"true" doc:"Node name or id."`
	}
}

func (d *Deps) RegisterCommons(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "commons-indices",
		Method:      http.MethodPost,
		Path:        "/commons/indices",
		Summary:     "List index names",
		Tags:        []string{"commons"},
	}, func(ctx context.Context, in *CommonsHostIn) (*Output[List[string]], error) {
		return transformListResp(ctx, d, in.Body,
			func(c context.Context, t elastic.Server) (elastic.Response, error) { return d.Client.GetIndices(c, t) },
			transform.CommonsIndices)
	})

	huma.Register(api, huma.Operation{
		OperationID: "commons-nodes",
		Method:      http.MethodPost,
		Path:        "/commons/nodes",
		Summary:     "List node names",
		Tags:        []string{"commons"},
	}, func(ctx context.Context, in *CommonsHostIn) (*Output[List[string]], error) {
		return transformListResp(ctx, d, in.Body,
			func(c context.Context, t elastic.Server) (elastic.Response, error) { return d.Client.GetNodes(c, t) },
			transform.CommonsNodes)
	})

	huma.Register(api, huma.Operation{
		OperationID: "commons-get-index-settings",
		Method:      http.MethodPost,
		Path:        "/commons/get_index_settings",
		Summary:     "Get index settings",
		Tags:        []string{"commons"},
	}, func(ctx context.Context, in *CommonsIndexIn) (*RawOutput, error) {
		return d.passthrough(ctx, in.Body.HostBody, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.GetIndexSettings(c, in.Body.Index, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "commons-get-index-mapping",
		Method:      http.MethodPost,
		Path:        "/commons/get_index_mapping",
		Summary:     "Get index mapping",
		Tags:        []string{"commons"},
	}, func(ctx context.Context, in *CommonsIndexIn) (*RawOutput, error) {
		return d.passthrough(ctx, in.Body.HostBody, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.GetIndexMapping(c, in.Body.Index, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "commons-get-node-stats",
		Method:      http.MethodPost,
		Path:        "/commons/get_node_stats",
		Summary:     "Get node stats",
		Tags:        []string{"commons"},
	}, func(ctx context.Context, in *CommonsNodeIn) (*RawOutput, error) {
		return d.passthrough(ctx, in.Body.HostBody, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.NodeStats(c, in.Body.Node, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "commons-get-index-stats",
		Method:      http.MethodPost,
		Path:        "/commons/get_index_stats",
		Summary:     "Get index stats",
		Tags:        []string{"commons"},
	}, func(ctx context.Context, in *CommonsIndexIn) (*RawOutput, error) {
		return d.passthrough(ctx, in.Body.HostBody, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.IndexStats(c, in.Body.Index, t)
		})
	})
}
