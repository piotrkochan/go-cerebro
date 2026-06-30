package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/elastic"
	"github.com/lmenezes/cerebro/internal/transform"
	"golang.org/x/sync/errgroup"
)

type ClusterChangesIn struct {
	ClusterPath
}

func (d *Deps) RegisterClusterChanges(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "cluster-changes",
		Method:      http.MethodGet,
		Path:        "/cluster_changes",
		Summary:     "Get cluster changes snapshot",
		Description: "Returns the current index and node names plus the cluster name, used by the UI to detect cluster topology changes.",
		Tags:        []string{"cluster"},
	}, func(ctx context.Context, in *ClusterChangesIn) (*Output[transform.Changes], error) {
		t, err := clusterTarget(ctx)
		if err != nil {
			return failMsg[transform.Changes](400, err.Error())
		}
		var aliases, nodesInfo, state elastic.Response
		g, gctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			r, err := d.Client.ExecuteRequest(gctx, http.MethodGet, "_aliases", nil, t)
			aliases = r
			return err
		})
		g.Go(func() error {
			r, err := d.Client.ExecuteRequest(gctx, http.MethodGet, "_nodes/transport", nil, t)
			nodesInfo = r
			return err
		})
		g.Go(func() error {
			r, err := d.Client.ExecuteRequest(gctx, http.MethodGet, "_cluster/state/blocks", nil, t)
			state = r
			return err
		})
		if err := g.Wait(); err != nil {
			return failMsg[transform.Changes](500, err.Error())
		}
		if e := firstError([]elastic.Response{aliases, nodesInfo, state}); e != nil {
			return fail[transform.Changes](e.Status, e.Body)
		}
		body, err := transform.ClusterChanges(aliases.Body, nodesInfo.Body, state.Body)
		if err != nil {
			return failMsg[transform.Changes](500, err.Error())
		}
		return ok(200, body)
	})
}
