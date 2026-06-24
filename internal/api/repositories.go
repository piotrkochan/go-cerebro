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
	Body HostBody
}

type RepositoriesCreateIn struct {
	Body struct {
		HostBody
		Name     string          `json:"name" required:"true" doc:"Repository name."`
		Type     string          `json:"type" required:"true" doc:"Repository type (fs, s3, url, ...)."`
		Settings json.RawMessage `json:"settings" required:"true" doc:"Repository settings, type-specific."`
	}
}

type RepositoriesDeleteIn struct {
	Body struct {
		HostBody
		Name string `json:"name" required:"true" doc:"Repository name."`
	}
}

func (d *Deps) RegisterRepositories(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "repositories-list",
		Method:      http.MethodPost,
		Path:        "/repositories",
		Summary:     "List snapshot repositories",
		Tags:        []string{"repositories"},
	}, func(ctx context.Context, in *RepositoriesListIn) (*Output[List[transform.Repository]], error) {
		t, err := d.resolveTarget(httpRequest(ctx), in.Body)
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
		Method:      http.MethodPost,
		Path:        "/repositories/create",
		Summary:     "Create a snapshot repository",
		Tags:        []string{"repositories"},
	}, func(ctx context.Context, in *RepositoriesCreateIn) (*RawOutput, error) {
		return d.passthrough(ctx, in.Body.HostBody, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.CreateRepository(c, in.Body.Name, in.Body.Type, in.Body.Settings, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "repositories-delete",
		Method:      http.MethodPost,
		Path:        "/repositories/delete",
		Summary:     "Delete a snapshot repository",
		Tags:        []string{"repositories"},
	}, func(ctx context.Context, in *RepositoriesDeleteIn) (*RawOutput, error) {
		return d.passthrough(ctx, in.Body.HostBody, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.DeleteRepository(c, in.Body.Name, t)
		})
	})
}
