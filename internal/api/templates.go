package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/elastic"
	"github.com/lmenezes/cerebro/internal/transform"
)

type TemplatesHostIn struct {
	Body HostBody
}

type TemplatesDeleteIn struct {
	Body struct {
		HostBody
		Name string `json:"name" required:"true"`
	}
}

type TemplatesCreateIn struct {
	Body struct {
		HostBody
		Name     string          `json:"name" required:"true"`
		Template json.RawMessage `json:"template" required:"true"`
	}
}

func (d *Deps) RegisterTemplates(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "templates-list",
		Method:      http.MethodPost,
		Path:        "/templates",
		Tags:        []string{"templates"},
	}, func(ctx context.Context, in *TemplatesHostIn) (*Output[List[transform.Template]], error) {
		return transformListResp(ctx, d, in.Body,
			func(c context.Context, t elastic.Server) (elastic.Response, error) {
				return d.Client.GetTemplates(c, t)
			},
			transform.Templates)
	})

	huma.Register(api, huma.Operation{
		OperationID: "templates-delete",
		Method:      http.MethodPost,
		Path:        "/templates/delete",
		Tags:        []string{"templates"},
	}, func(ctx context.Context, in *TemplatesDeleteIn) (*RawOutput, error) {
		return d.passthrough(ctx, in.Body.HostBody, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.DeleteTemplate(c, in.Body.Name, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "templates-create",
		Method:      http.MethodPost,
		Path:        "/templates/create",
		Tags:        []string{"templates"},
	}, func(ctx context.Context, in *TemplatesCreateIn) (*RawOutput, error) {
		return d.passthrough(ctx, in.Body.HostBody, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.CreateTemplate(c, in.Body.Name, in.Body.Template, t)
		})
	})
}
