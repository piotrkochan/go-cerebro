package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/elastic"
	"github.com/lmenezes/cerebro/internal/transform"
	"golang.org/x/sync/errgroup"
)

type SnapshotsGetIn struct {
	ClusterPath
}

type SnapshotsLoadIn struct {
	ClusterPath
	Repository string `path:"repository" doc:"Repository name."`
}

type SnapshotsDeleteIn struct {
	ClusterPath
	Repository string `path:"repository" doc:"Repository name."`
	Snapshot   string `path:"snapshot" doc:"Snapshot name."`
}

type SnapshotsCreateIn struct {
	ClusterPath
	Repository string `path:"repository" doc:"Repository name."`
	Snapshot   string `path:"snapshot" doc:"Snapshot name."`
	Body       struct {
		Indices            []string `json:"indices,omitempty"`
		IgnoreUnavailable  bool     `json:"ignoreUnavailable"`
		IncludeGlobalState bool     `json:"includeGlobalState"`
	}
}

type SnapshotsRestoreIn struct {
	ClusterPath
	Repository string `path:"repository" doc:"Repository name."`
	Snapshot   string `path:"snapshot" doc:"Snapshot name."`
	Body       struct {
		RenamePattern      string   `json:"renamePattern,omitempty"`
		RenameReplacement  string   `json:"renameReplacement,omitempty"`
		IgnoreUnavailable  bool     `json:"ignoreUnavailable"`
		IncludeAliases     bool     `json:"includeAliases"`
		IncludeGlobalState bool     `json:"includeGlobalState"`
		Indices            []string `json:"indices,omitempty"`
	}
}

func (d *Deps) RegisterSnapshots(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "snapshots-get",
		Method:      http.MethodGet,
		Path:        "/snapshots",
		Tags:        []string{"snapshots"},
	}, func(ctx context.Context, in *SnapshotsGetIn) (*RawOutput, error) {
		t, err := clusterTarget(ctx)
		if err != nil {
			return failMsg[RawResponse](400, err.Error())
		}
		var indices, repos elastic.Response
		g, gctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			r, err := d.Client.GetIndices(gctx, t)
			indices = r
			return err
		})
		g.Go(func() error {
			r, err := d.Client.GetRepositories(gctx, t)
			repos = r
			return err
		})
		if err := g.Wait(); err != nil {
			return failMsg[RawResponse](500, err.Error())
		}
		out := map[string]any{
			"indices":      transform.SnapshotIndices(indices.Body),
			"repositories": transform.SnapshotRepositories(repos.Body),
		}
		body, _ := json.Marshal(out)
		return raw(200, body)
	})

	huma.Register(api, huma.Operation{
		OperationID: "snapshots-load",
		Method:      http.MethodGet,
		Path:        "/snapshots/{repository}",
		Tags:        []string{"snapshots"},
	}, func(ctx context.Context, in *SnapshotsLoadIn) (*RawOutput, error) {
		return transformRawResp(ctx, d,
			func(c context.Context, t elastic.Server) (elastic.Response, error) {
				return d.Client.GetSnapshots(c, in.Repository, t)
			},
			transform.SnapshotsList)
	})

	huma.Register(api, huma.Operation{
		OperationID: "snapshots-delete",
		Method:      http.MethodDelete,
		Path:        "/snapshots/{repository}/{snapshot}",
		Tags:        []string{"snapshots"},
	}, func(ctx context.Context, in *SnapshotsDeleteIn) (*RawOutput, error) {
		return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.DeleteSnapshot(c, in.Repository, in.Snapshot, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "snapshots-create",
		Method:      http.MethodPut,
		Path:        "/snapshots/{repository}/{snapshot}",
		Tags:        []string{"snapshots"},
	}, func(ctx context.Context, in *SnapshotsCreateIn) (*RawOutput, error) {
		var indices *string
		if len(in.Body.Indices) > 0 {
			joined := strings.Join(in.Body.Indices, ",")
			indices = &joined
		}
		return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.CreateSnapshot(c, in.Repository, in.Snapshot, in.Body.IgnoreUnavailable, in.Body.IncludeGlobalState, indices, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "snapshots-restore",
		Method:      http.MethodPost,
		Path:        "/snapshots/{repository}/{snapshot}/restore",
		Tags:        []string{"snapshots"},
	}, func(ctx context.Context, in *SnapshotsRestoreIn) (*RawOutput, error) {
		var indices *string
		if len(in.Body.Indices) > 0 {
			joined := strings.Join(in.Body.Indices, ",")
			indices = &joined
		}
		var rp, rr *string
		if in.Body.RenamePattern != "" {
			rp = &in.Body.RenamePattern
		}
		if in.Body.RenameReplacement != "" {
			rr = &in.Body.RenameReplacement
		}
		return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.RestoreSnapshot(c, in.Repository, in.Snapshot, rp, rr,
				in.Body.IgnoreUnavailable, in.Body.IncludeAliases, in.Body.IncludeGlobalState, indices, t)
		})
	})
}
