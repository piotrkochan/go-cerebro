package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/elastic"
	"github.com/lmenezes/cerebro/internal/transform"
)

type TemplatesListIn struct {
	ClusterPath
}

type TemplateResourceIn struct {
	ClusterPath
	Kind string `path:"kind" enum:"index,component,legacy" doc:"Template kind."`
	Name string `path:"name" doc:"Template name."`
}

type TemplatePutIn struct {
	TemplateResourceIn
	Body json.RawMessage `required:"true" doc:"Raw template definition."`
}

func (d *Deps) RegisterTemplates(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "templates-list",
		Method:      http.MethodGet,
		Path:        "/templates",
		Tags:        []string{"templates"},
	}, func(ctx context.Context, in *TemplatesListIn) (*Output[List[transform.TemplateSummary]], error) {
		t, err := clusterTarget(ctx)
		if err != nil {
			return failMsg[List[transform.TemplateSummary]](400, err.Error())
		}
		indexResp, err := d.Client.GetComposableIndexTemplates(ctx, t)
		if err != nil {
			return failMsg[List[transform.TemplateSummary]](500, err.Error())
		}
		if indexResp.IsSuccess() {
			items := transform.ComposableIndexTemplateSummaries(indexResp.Body)
			componentResp, err := d.Client.GetComponentTemplates(ctx, t)
			if err != nil {
				return failMsg[List[transform.TemplateSummary]](500, err.Error())
			}
			if !componentResp.IsSuccess() {
				return fail[List[transform.TemplateSummary]](componentResp.Status, componentResp.Body)
			}
			items = append(items, transform.ComponentTemplateSummaries(componentResp.Body)...)
			if legacyResp, err := d.Client.GetTemplates(ctx, t); err == nil && legacyResp.IsSuccess() {
				items = append(items, transform.TemplateSummaries(legacyResp.Body)...)
			}
			return okList(indexResp.Status, items)
		}
		legacyResp, err := d.Client.GetTemplates(ctx, t)
		if err != nil {
			return failMsg[List[transform.TemplateSummary]](500, err.Error())
		}
		if !legacyResp.IsSuccess() {
			return fail[List[transform.TemplateSummary]](legacyResp.Status, legacyResp.Body)
		}
		return okList(legacyResp.Status, transform.TemplateSummaries(legacyResp.Body))
	})

	huma.Register(api, huma.Operation{
		OperationID: "templates-get",
		Method:      http.MethodGet,
		Path:        "/templates/{kind}/{name}",
		Tags:        []string{"templates"},
	}, func(ctx context.Context, in *TemplateResourceIn) (*Output[transform.Template], error) {
		t, err := clusterTarget(ctx)
		if err != nil {
			return failMsg[transform.Template](400, err.Error())
		}
		var resp elastic.Response
		switch normalizeTemplateKind(in.Kind) {
		case "component":
			resp, err = d.Client.GetComponentTemplate(ctx, in.Name, t)
		case "legacy":
			resp, err = d.Client.GetTemplate(ctx, in.Name, t)
		default:
			resp, err = d.Client.GetComposableIndexTemplate(ctx, in.Name, t)
		}
		if err != nil {
			return failMsg[transform.Template](500, err.Error())
		}
		if !resp.IsSuccess() {
			return fail[transform.Template](resp.Status, resp.Body)
		}
		template, found := templateDetail(normalizeTemplateKind(in.Kind), resp.Body, in.Name)
		if !found {
			return failMsg[transform.Template](404, "template not found")
		}
		return ok(resp.Status, template)
	})

	huma.Register(api, huma.Operation{
		OperationID: "templates-delete",
		Method:      http.MethodDelete,
		Path:        "/templates/{kind}/{name}",
		Tags:        []string{"templates"},
	}, func(ctx context.Context, in *TemplateResourceIn) (*RawOutput, error) {
		return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			switch normalizeTemplateKind(in.Kind) {
			case "component":
				return d.Client.DeleteComponentTemplate(c, in.Name, t)
			case "legacy":
				return d.Client.DeleteTemplate(c, in.Name, t)
			default:
				return d.Client.DeleteComposableIndexTemplate(c, in.Name, t)
			}
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "templates-create",
		Method:      http.MethodPut,
		Path:        "/templates/{kind}/{name}",
		Tags:        []string{"templates"},
	}, func(ctx context.Context, in *TemplatePutIn) (*RawOutput, error) {
		return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			switch normalizeTemplateKind(in.Kind) {
			case "component":
				return d.Client.CreateComponentTemplate(c, in.Name, in.Body, t)
			case "legacy":
				return d.Client.CreateTemplate(c, in.Name, in.Body, t)
			default:
				return d.Client.CreateComposableIndexTemplate(c, in.Name, in.Body, t)
			}
		})
	})
}

func templateDetail(kind string, body json.RawMessage, name string) (transform.Template, bool) {
	switch normalizeTemplateKind(kind) {
	case "component":
		return transform.ComponentTemplate(body, name)
	case "legacy":
		return transform.LegacyTemplate(body, name)
	default:
		return transform.ComposableIndexTemplate(body, name)
	}
}

func normalizeTemplateKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "index", "component", "legacy":
		return strings.ToLower(strings.TrimSpace(kind))
	default:
		return ""
	}
}
