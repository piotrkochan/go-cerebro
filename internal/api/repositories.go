package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/elastic"
	"github.com/lmenezes/cerebro/internal/transform"
)

type RepositoriesListIn struct {
	ClusterPath
}

type RepositoriesCreateIn struct {
	ClusterPath
	Name string `path:"name" doc:"Repository name."`
	Body struct {
		Type     string          `json:"type" required:"true" doc:"Repository type (fs, s3, url, ...)."`
		Settings json.RawMessage `json:"settings" required:"true" doc:"Repository settings, type-specific."`
	}
}

type RepositoriesDeleteIn struct {
	ClusterPath
	Name string `path:"name" doc:"Repository name."`
}

func (d *Deps) RegisterRepositories(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "repositories-list",
		Method:      http.MethodGet,
		Path:        "/repositories",
		Summary:     "List snapshot repositories",
		Tags:        []string{"repositories"},
	}, func(ctx context.Context, in *RepositoriesListIn) (*Output[List[transform.Repository]], error) {
		t, err := clusterTarget(ctx)
		if err != nil {
			return failMsg[List[transform.Repository]](400, err.Error())
		}
		resp, err := d.Client.GetRepositories(ctx, t)
		if err != nil {
			return failMsg[List[transform.Repository]](500, err.Error())
		}
		if !resp.IsSuccess() {
			return fail[List[transform.Repository]](resp.Status, resp.Body)
		}
		body, err := transform.Repositories(resp.Body)
		if err != nil {
			return failMsg[List[transform.Repository]](500, err.Error())
		}
		return okList(resp.Status, body)
	})

	huma.Register(api, huma.Operation{
		OperationID: "repositories-create",
		Method:      http.MethodPut,
		Path:        "/repositories/{name}",
		Summary:     "Create a snapshot repository",
		Tags:        []string{"repositories"},
	}, func(ctx context.Context, in *RepositoriesCreateIn) (*RawOutput, error) {
		return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.CreateRepository(c, in.Name, in.Body.Type, in.Body.Settings, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "repositories-delete",
		Method:      http.MethodDelete,
		Path:        "/repositories/{name}",
		Summary:     "Delete a snapshot repository",
		Tags:        []string{"repositories"},
	}, func(ctx context.Context, in *RepositoriesDeleteIn) (*RawOutput, error) {
		return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.DeleteRepository(c, in.Name, t)
		})
	})
}
