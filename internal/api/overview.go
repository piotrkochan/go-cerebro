package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/elastic"
	"github.com/lmenezes/cerebro/internal/transform"
	"golang.org/x/sync/errgroup"
)

type OverviewIn struct {
	Body HostBody
}

type OverviewShardKindIn struct {
	Body struct {
		HostBody
		Kind string `json:"kind" required:"true" doc:"Allocation kind to keep enabled, e.g. \"primaries\" or \"none\"."`
	}
}

type OverviewIndicesIn struct {
	Body struct {
		HostBody
		Indices string `json:"indices" required:"true" doc:"Comma-separated index names."`
	}
}

type OverviewShardStatsIn struct {
	Body struct {
		HostBody
		Index string `json:"index" required:"true" doc:"Index name."`
		Shard int    `json:"shard" required:"true" doc:"Shard number."`
		Node  string `json:"node" required:"true" doc:"Node id the shard is allocated on."`
	}
}

type OverviewRelocateIn struct {
	Body struct {
		HostBody
		Index string `json:"index" required:"true" doc:"Index name."`
		Shard int    `json:"shard" required:"true" doc:"Shard number."`
		From  string `json:"from" required:"true" doc:"Source node."`
		To    string `json:"to" required:"true" doc:"Destination node."`
	}
}

func (d *Deps) RegisterOverview(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "overview",
		Method:      http.MethodPost,
		Path:        "/overview",
		Summary:     "Get cluster overview",
		Description: "Returns the cluster dashboard payload: health, indices with shard routing, and nodes.",
		Tags:        []string{"overview"},
	}, func(ctx context.Context, in *OverviewIn) (*Output[transform.Overview], error) {
		t, err := d.resolveTarget(httpRequest(ctx), in.Body)
		if err != nil {
			return failMsg[transform.Overview](400, err.Error())
		}
		paths := []string{
			"_cluster/state/master_node,routing_table,blocks,metadata",
			"_nodes/stats/jvm,fs,os,process?human=true",
			"_stats/docs,store?ignore_unavailable=true",
			"_cluster/settings",
			"_aliases",
			"_cluster/health",
			"_nodes/_all/os,jvm?human=true",
		}
		responses := make([]elastic.Response, len(paths))
		g, gctx := errgroup.WithContext(ctx)
		for i, p := range paths {
			g.Go(func() error {
				r, err := d.Client.ExecuteRequest(gctx, http.MethodGet, p, nil, t)
				responses[i] = r
				return err
			})
		}
		if err := g.Wait(); err != nil {
			return failMsg[transform.Overview](500, err.Error())
		}
		if e := firstError(responses); e != nil {
			return fail[transform.Overview](e.Status, e.Body)
		}
		body, err := transform.ClusterOverview(
			responses[0].Body, responses[1].Body, responses[2].Body, responses[3].Body,
			responses[4].Body, responses[5].Body, responses[6].Body,
		)
		if err != nil {
			return failMsg[transform.Overview](500, err.Error())
		}
		return ok(200, body)
	})

	huma.Register(api, huma.Operation{
		OperationID: "overview-disable-shard-allocation",
		Method:      http.MethodPost,
		Path:        "/overview/disable_shard_allocation",
		Summary:     "Disable shard allocation",
		Tags:        []string{"overview"},
	}, func(ctx context.Context, in *OverviewShardKindIn) (*RawOutput, error) {
		return d.passthrough(ctx, in.Body.HostBody, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.DisableShardAllocation(c, in.Body.Kind, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "overview-enable-shard-allocation",
		Method:      http.MethodPost,
		Path:        "/overview/enable_shard_allocation",
		Summary:     "Enable shard allocation",
		Tags:        []string{"overview"},
	}, func(ctx context.Context, in *OverviewIn) (*RawOutput, error) {
		return d.passthrough(ctx, in.Body, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.EnableShardAllocation(c, t)
		})
	})

	indexAction := func(opID, path, summary string, fn func(c context.Context, t elastic.Server, indices string) (elastic.Response, error)) {
		huma.Register(api, huma.Operation{
			OperationID: opID,
			Method:      http.MethodPost,
			Path:        path,
			Summary:     summary,
			Tags:        []string{"overview"},
		}, func(ctx context.Context, in *OverviewIndicesIn) (*RawOutput, error) {
			return d.passthrough(ctx, in.Body.HostBody, func(c context.Context, t elastic.Server) (elastic.Response, error) {
				return fn(c, t, in.Body.Indices)
			})
		})
	}
	indexAction("overview-close-indices", "/overview/close_indices", "Close indices",
		func(c context.Context, t elastic.Server, idx string) (elastic.Response, error) {
			return d.Client.CloseIndex(c, idx, t)
		})
	indexAction("overview-open-indices", "/overview/open_indices", "Open indices",
		func(c context.Context, t elastic.Server, idx string) (elastic.Response, error) {
			return d.Client.OpenIndex(c, idx, t)
		})
	indexAction("overview-force-merge", "/overview/force_merge", "Force merge indices",
		func(c context.Context, t elastic.Server, idx string) (elastic.Response, error) {
			return d.Client.ForceMerge(c, idx, t)
		})
	indexAction("overview-clear-indices-cache", "/overview/clear_indices_cache", "Clear indices cache",
		func(c context.Context, t elastic.Server, idx string) (elastic.Response, error) {
			return d.Client.ClearIndexCache(c, idx, t)
		})
	indexAction("overview-refresh-indices", "/overview/refresh_indices", "Refresh indices",
		func(c context.Context, t elastic.Server, idx string) (elastic.Response, error) {
			return d.Client.RefreshIndex(c, idx, t)
		})
	indexAction("overview-flush-indices", "/overview/flush_indices", "Flush indices",
		func(c context.Context, t elastic.Server, idx string) (elastic.Response, error) {
			return d.Client.FlushIndex(c, idx, t)
		})
	indexAction("overview-delete-indices", "/overview/delete_indices", "Delete indices",
		func(c context.Context, t elastic.Server, idx string) (elastic.Response, error) {
			return d.Client.DeleteIndex(c, idx, t)
		})

	huma.Register(api, huma.Operation{
		OperationID: "overview-shard-stats",
		Method:      http.MethodPost,
		Path:        "/overview/get_shard_stats",
		Summary:     "Get shard stats",
		Description: "Returns stats (or recovery info) for one shard of an index on a given node.",
		Tags:        []string{"overview"},
	}, func(ctx context.Context, in *OverviewShardStatsIn) (*RawOutput, error) {
		t, err := d.resolveTarget(httpRequest(ctx), in.Body.HostBody)
		if err != nil {
			return failMsg[RawResponse](400, err.Error())
		}
		var stats, recovery elastic.Response
		g, gctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			r, err := d.Client.GetShardStats(gctx, in.Body.Index, t)
			stats = r
			return err
		})
		g.Go(func() error {
			r, err := d.Client.GetIndexRecovery(gctx, in.Body.Index, t)
			recovery = r
			return err
		})
		if err := g.Wait(); err != nil {
			return failMsg[RawResponse](500, err.Error())
		}
		if e := firstError([]elastic.Response{stats, recovery}); e != nil {
			return fail[RawResponse](e.Status, e.Body)
		}
		return raw(200, transform.ShardStats(in.Body.Index, in.Body.Node, in.Body.Shard, stats.Body, recovery.Body))
	})

	huma.Register(api, huma.Operation{
		OperationID: "overview-relocate-shard",
		Method:      http.MethodPost,
		Path:        "/overview/relocate_shard",
		Summary:     "Relocate a shard",
		Tags:        []string{"overview"},
	}, func(ctx context.Context, in *OverviewRelocateIn) (*RawOutput, error) {
		return d.passthrough(ctx, in.Body.HostBody, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.RelocateShard(c, in.Body.Shard, in.Body.Index, in.Body.From, in.Body.To, t)
		})
	})
}
