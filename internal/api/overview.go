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
	ClusterPath
}

type OverviewShardKindIn struct {
	ClusterPath
	Body struct {
		Kind string `json:"kind" required:"true" doc:"Allocation kind to keep enabled, e.g. \"primaries\" or \"none\"."`
	}
}

type OverviewIndicesIn struct {
	ClusterPath
	Indices string `path:"indices" doc:"Comma-separated index names."`
}

type OverviewShardStatsIn struct {
	ClusterPath
	Index string `path:"index" doc:"Index name."`
	Shard int    `path:"shard" doc:"Shard number."`
	Node  string `query:"node" required:"true" doc:"Node id the shard is allocated on."`
}

type OverviewRelocateIn struct {
	ClusterPath
	Index string `path:"index" doc:"Index name."`
	Shard int    `path:"shard" doc:"Shard number."`
	Body  struct {
		From string `json:"from" required:"true" doc:"Source node."`
		To   string `json:"to" required:"true" doc:"Destination node."`
	}
}

func (d *Deps) RegisterOverview(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "overview",
		Method:      http.MethodGet,
		Path:        "/overview",
		Summary:     "Get cluster overview",
		Description: "Returns the cluster dashboard payload: health, indices with shard routing, and nodes.",
		Tags:        []string{"overview"},
	}, func(ctx context.Context, in *OverviewIn) (*Output[transform.Overview], error) {
		t, err := clusterTarget(ctx)
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
		Method:      http.MethodPut,
		Path:        "/overview/shard_allocation",
		Summary:     "Disable shard allocation",
		Tags:        []string{"overview"},
	}, func(ctx context.Context, in *OverviewShardKindIn) (*RawOutput, error) {
		return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.DisableShardAllocation(c, in.Body.Kind, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "overview-enable-shard-allocation",
		Method:      http.MethodDelete,
		Path:        "/overview/shard_allocation",
		Summary:     "Enable shard allocation",
		Tags:        []string{"overview"},
	}, func(ctx context.Context, in *OverviewIn) (*RawOutput, error) {
		return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.EnableShardAllocation(c, t)
		})
	})

	indexAction := func(opID, method, path, summary string, fn func(c context.Context, t elastic.Server, indices string) (elastic.Response, error)) {
		huma.Register(api, huma.Operation{
			OperationID: opID,
			Method:      method,
			Path:        path,
			Summary:     summary,
			Tags:        []string{"overview"},
		}, func(ctx context.Context, in *OverviewIndicesIn) (*RawOutput, error) {
			return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
				return fn(c, t, in.Indices)
			})
		})
	}
	indexAction("overview-close-indices", http.MethodPost, "/overview/indices/{indices}/close", "Close indices",
		func(c context.Context, t elastic.Server, idx string) (elastic.Response, error) {
			return d.Client.CloseIndex(c, idx, t)
		})
	indexAction("overview-open-indices", http.MethodPost, "/overview/indices/{indices}/open", "Open indices",
		func(c context.Context, t elastic.Server, idx string) (elastic.Response, error) {
			return d.Client.OpenIndex(c, idx, t)
		})
	indexAction("overview-force-merge", http.MethodPost, "/overview/indices/{indices}/forcemerge", "Force merge indices",
		func(c context.Context, t elastic.Server, idx string) (elastic.Response, error) {
			return d.Client.ForceMerge(c, idx, t)
		})
	indexAction("overview-clear-indices-cache", http.MethodPost, "/overview/indices/{indices}/cache/clear", "Clear indices cache",
		func(c context.Context, t elastic.Server, idx string) (elastic.Response, error) {
			return d.Client.ClearIndexCache(c, idx, t)
		})
	indexAction("overview-refresh-indices", http.MethodPost, "/overview/indices/{indices}/refresh", "Refresh indices",
		func(c context.Context, t elastic.Server, idx string) (elastic.Response, error) {
			return d.Client.RefreshIndex(c, idx, t)
		})
	indexAction("overview-flush-indices", http.MethodPost, "/overview/indices/{indices}/flush", "Flush indices",
		func(c context.Context, t elastic.Server, idx string) (elastic.Response, error) {
			return d.Client.FlushIndex(c, idx, t)
		})
	indexAction("overview-delete-indices", http.MethodDelete, "/overview/indices/{indices}", "Delete indices",
		func(c context.Context, t elastic.Server, idx string) (elastic.Response, error) {
			return d.Client.DeleteIndex(c, idx, t)
		})

	huma.Register(api, huma.Operation{
		OperationID: "overview-shard-stats",
		Method:      http.MethodGet,
		Path:        "/overview/indices/{index}/shards/{shard}/stats",
		Summary:     "Get shard stats",
		Description: "Returns stats (or recovery info) for one shard of an index on a given node.",
		Tags:        []string{"overview"},
	}, func(ctx context.Context, in *OverviewShardStatsIn) (*RawOutput, error) {
		t, err := clusterTarget(ctx)
		if err != nil {
			return failMsg[RawResponse](400, err.Error())
		}
		var stats, recovery elastic.Response
		g, gctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			r, err := d.Client.GetShardStats(gctx, in.Index, t)
			stats = r
			return err
		})
		g.Go(func() error {
			r, err := d.Client.GetIndexRecovery(gctx, in.Index, t)
			recovery = r
			return err
		})
		if err := g.Wait(); err != nil {
			return failMsg[RawResponse](500, err.Error())
		}
		if e := firstError([]elastic.Response{stats, recovery}); e != nil {
			return fail[RawResponse](e.Status, e.Body)
		}
		return raw(200, transform.ShardStats(in.Index, in.Node, in.Shard, stats.Body, recovery.Body))
	})

	huma.Register(api, huma.Operation{
		OperationID: "overview-relocate-shard",
		Method:      http.MethodPost,
		Path:        "/overview/indices/{index}/shards/{shard}/relocation",
		Summary:     "Relocate a shard",
		Tags:        []string{"overview"},
	}, func(ctx context.Context, in *OverviewRelocateIn) (*RawOutput, error) {
		return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.RelocateShard(c, in.Shard, in.Index, in.Body.From, in.Body.To, t)
		})
	})
}
