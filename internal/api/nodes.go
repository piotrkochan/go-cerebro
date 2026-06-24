package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/elastic"
	"github.com/lmenezes/cerebro/internal/transform"
	"golang.org/x/sync/errgroup"
)

type NodesIn struct {
	Body HostBody
}

func (d *Deps) RegisterNodes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "nodes",
		Method:      http.MethodPost,
		Path:        "/nodes",
		Summary:     "List cluster nodes",
		Description: "Returns one entry per node with roles, heap, disk, CPU and uptime information.",
		Tags:        []string{"nodes"},
	}, func(ctx context.Context, in *NodesIn) (*Output[List[transform.Node]], error) {
		t, err := d.resolveTarget(httpRequest(ctx), in.Body)
		if err != nil {
			return failMsg[List[transform.Node]](400, err.Error())
		}
		var (
			info, stats, master elastic.Response
		)
		g, gctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			r, err := d.Client.Nodes(gctx, []string{"jvm", "os"}, t)
			info = r
			return err
		})
		g.Go(func() error {
			r, err := d.Client.NodesStats(gctx, []string{"jvm", "fs", "os", "process"}, t)
			stats = r
			return err
		})
		g.Go(func() error {
			r, err := d.Client.CatMaster(gctx, t)
			master = r
			return err
		})
		if err := g.Wait(); err != nil {
			return failMsg[List[transform.Node]](500, err.Error())
		}
		if e := firstError([]elastic.Response{info, stats, master}); e != nil {
			return fail[List[transform.Node]](e.Status, e.Body)
		}
		return okList(200, transform.Nodes(info.Body, stats.Body, master.Body))
	})
}
