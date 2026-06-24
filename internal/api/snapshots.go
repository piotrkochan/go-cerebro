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
	Body HostBody
}

type SnapshotsLoadIn struct {
	Body struct {
		HostBody
		Repository string `json:"repository" required:"true"`
	}
}

type SnapshotsDeleteIn struct {
	Body struct {
		HostBody
		Repository string `json:"repository" required:"true"`
		Snapshot   string `json:"snapshot" required:"true"`
	}
}

type SnapshotsCreateIn struct {
	Body struct {
		HostBody
		Repository         string   `json:"repository" required:"true"`
		Snapshot           string   `json:"snapshot" required:"true"`
		Indices            []string `json:"indices,omitempty"`
		IgnoreUnavailable  bool     `json:"ignoreUnavailable"`
		IncludeGlobalState bool     `json:"includeGlobalState"`
	}
}

type SnapshotsRestoreIn struct {
	Body struct {
		HostBody
		Repository         string   `json:"repository" required:"true"`
		Snapshot           string   `json:"snapshot" required:"true"`
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
		Method:      http.MethodPost,
		Path:        "/snapshots",
		Tags:        []string{"snapshots"},
	}, func(ctx context.Context, in *SnapshotsGetIn) (*RawOutput, error) {
		t, err := d.resolveTarget(httpRequest(ctx), in.Body)
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
		Method:      http.MethodPost,
		Path:        "/snapshots/load",
		Tags:        []string{"snapshots"},
	}, func(ctx context.Context, in *SnapshotsLoadIn) (*RawOutput, error) {
		return transformRawResp(ctx, d, in.Body.HostBody,
			func(c context.Context, t elastic.Server) (elastic.Response, error) {
				return d.Client.GetSnapshots(c, in.Body.Repository, t)
			},
			transform.SnapshotsList)
	})

	huma.Register(api, huma.Operation{
		OperationID: "snapshots-delete",
		Method:      http.MethodPost,
		Path:        "/snapshots/delete",
		Tags:        []string{"snapshots"},
	}, func(ctx context.Context, in *SnapshotsDeleteIn) (*RawOutput, error) {
		return d.passthrough(ctx, in.Body.HostBody, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.DeleteSnapshot(c, in.Body.Repository, in.Body.Snapshot, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "snapshots-create",
		Method:      http.MethodPost,
		Path:        "/snapshots/create",
		Tags:        []string{"snapshots"},
	}, func(ctx context.Context, in *SnapshotsCreateIn) (*RawOutput, error) {
		var indices *string
		if len(in.Body.Indices) > 0 {
			joined := strings.Join(in.Body.Indices, ",")
			indices = &joined
		}
		return d.passthrough(ctx, in.Body.HostBody, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.CreateSnapshot(c, in.Body.Repository, in.Body.Snapshot, in.Body.IgnoreUnavailable, in.Body.IncludeGlobalState, indices, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "snapshots-restore",
		Method:      http.MethodPost,
		Path:        "/snapshots/restore",
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
		return d.passthrough(ctx, in.Body.HostBody, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.RestoreSnapshot(c, in.Body.Repository, in.Body.Snapshot, rp, rr,
				in.Body.IgnoreUnavailable, in.Body.IncludeAliases, in.Body.IncludeGlobalState, indices, t)
		})
	})
}
