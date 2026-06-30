package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/elastic"
	"github.com/lmenezes/cerebro/internal/transform"
	"golang.org/x/sync/errgroup"
)

type DataStreamsHostIn struct {
	ClusterPath
}

type DataStreamNameIn struct {
	ClusterPath
	Name string `path:"name" doc:"Data stream name."`
}

type DataStreamLifecycleIn struct {
	ClusterPath
	Name string `path:"name" doc:"Data stream name."`
	Body struct {
		Lifecycle json.RawMessage `json:"lifecycle" required:"true" doc:"Data stream lifecycle body."`
	}
}

type DataStreamAttachILMIn struct {
	ClusterPath
	Name string `path:"name" doc:"Data stream name."`
	Body struct {
		Policy               string `json:"policy" required:"true" doc:"ILM policy name."`
		UpdateBackingIndices bool   `json:"update_backing_indices" doc:"Whether existing backing indices should get index.lifecycle.name."`
		Rollover             bool   `json:"rollover" doc:"Whether to rollover the data stream after updating the template."`
	}
}

type DataStreamDetachILMIn struct {
	ClusterPath
	Name                 string `path:"name" doc:"Data stream name."`
	UpdateBackingIndices bool   `query:"update_backing_indices" doc:"Whether existing backing indices should have index.lifecycle.name removed."`
}

type DataStreamAttachILMResult struct {
	DataStream            string `json:"data_stream"`
	Policy                string `json:"policy"`
	Template              string `json:"template"`
	TemplateUpdated       bool   `json:"template_updated"`
	BackingIndicesUpdated bool   `json:"backing_indices_updated"`
	RolledOver            bool   `json:"rolled_over"`
}

type DataStreamDetachILMResult struct {
	DataStream            string `json:"data_stream"`
	Template              string `json:"template"`
	TemplateUpdated       bool   `json:"template_updated"`
	BackingIndicesUpdated bool   `json:"backing_indices_updated"`
}

func (d *Deps) RegisterDataStreams(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "data-streams-list",
		Method:      http.MethodGet,
		Path:        "/data_streams",
		Summary:     "List data streams",
		Description: "Returns a lightweight data stream list. Use GET /data_streams/{name} for backing indices, stats and lifecycle details.",
		Tags:        []string{"data-streams"},
	}, func(ctx context.Context, in *DataStreamsHostIn) (*Output[transform.DataStreamsResult], error) {
		t, err := clusterTarget(ctx)
		if err != nil {
			return failMsg[transform.DataStreamsResult](http.StatusBadRequest, err.Error())
		}
		streams, err := d.Client.GetDataStreams(ctx, t)
		if err != nil {
			return failMsg[transform.DataStreamsResult](http.StatusInternalServerError, err.Error())
		}
		if streams.Status == http.StatusNotFound || streams.Status == http.StatusBadRequest {
			return ok(http.StatusOK, transform.DataStreamsUnsupported())
		}
		if !streams.IsSuccess() {
			return fail[transform.DataStreamsResult](streams.Status, streams.Body)
		}
		return ok(http.StatusOK, transform.DataStreamSummaries(streams.Body))
	})

	huma.Register(api, huma.Operation{
		OperationID: "data-streams-get",
		Method:      http.MethodGet,
		Path:        "/data_streams/{name}",
		Summary:     "Get data stream details",
		Description: "Returns backing indices, stats and lifecycle information for one data stream.",
		Tags:        []string{"data-streams"},
	}, func(ctx context.Context, in *DataStreamNameIn) (*Output[transform.DataStream], error) {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return failMsg[transform.DataStream](http.StatusBadRequest, "data stream name is required")
		}
		t, err := clusterTarget(ctx)
		if err != nil {
			return failMsg[transform.DataStream](http.StatusBadRequest, err.Error())
		}
		streams, err := d.Client.GetDataStream(ctx, name, t)
		if err != nil {
			return failMsg[transform.DataStream](http.StatusInternalServerError, err.Error())
		}
		if !streams.IsSuccess() {
			return fail[transform.DataStream](streams.Status, streams.Body)
		}
		target, found := dataStreamTarget(streams.Body, name)
		if !found {
			return failMsg[transform.DataStream](http.StatusNotFound, "data stream not found")
		}

		var stats, cat, ilm elastic.Response
		g, gctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			stats, _ = d.Client.GetDataStreamStats(gctx, name, t)
			return nil
		})
		g.Go(func() error {
			cat, _ = d.Client.GetDataStreamBackingIndices(gctx, target.BackingIndices, t)
			return nil
		})
		g.Go(func() error {
			ilm, _ = d.Client.GetDataStreamILM(gctx, target.BackingIndices, t)
			return nil
		})
		_ = g.Wait()
		detail, found := transform.DataStreamDetail(streams.Body, successfulBody(stats), successfulBody(cat), successfulBody(ilm), name)
		if !found {
			return failMsg[transform.DataStream](http.StatusNotFound, "data stream not found")
		}
		return ok(http.StatusOK, detail)
	})

	huma.Register(api, huma.Operation{
		OperationID: "data-streams-create",
		Method:      http.MethodPut,
		Path:        "/data_streams/{name}",
		Summary:     "Create data stream",
		Tags:        []string{"data-streams"},
	}, func(ctx context.Context, in *DataStreamNameIn) (*RawOutput, error) {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return failMsg[RawResponse](http.StatusBadRequest, "data stream name is required")
		}
		return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.CreateDataStream(c, name, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "data-streams-rollover",
		Method:      http.MethodPost,
		Path:        "/data_streams/{name}/rollover",
		Summary:     "Rollover data stream",
		Tags:        []string{"data-streams"},
	}, func(ctx context.Context, in *DataStreamNameIn) (*RawOutput, error) {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return failMsg[RawResponse](http.StatusBadRequest, "data stream name is required")
		}
		return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.RolloverDataStream(c, name, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "data-streams-delete",
		Method:      http.MethodDelete,
		Path:        "/data_streams/{name}",
		Summary:     "Delete data stream",
		Tags:        []string{"data-streams"},
	}, func(ctx context.Context, in *DataStreamNameIn) (*RawOutput, error) {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return failMsg[RawResponse](http.StatusBadRequest, "data stream name is required")
		}
		return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.DeleteDataStream(c, name, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "data-streams-update-lifecycle",
		Method:      http.MethodPut,
		Path:        "/data_streams/{name}/lifecycle",
		Summary:     "Update data stream lifecycle",
		Tags:        []string{"data-streams"},
	}, func(ctx context.Context, in *DataStreamLifecycleIn) (*RawOutput, error) {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return failMsg[RawResponse](http.StatusBadRequest, "data stream name is required")
		}
		if !json.Valid(in.Body.Lifecycle) {
			return failMsg[RawResponse](http.StatusBadRequest, "lifecycle must be valid JSON")
		}
		return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.PutDataStreamLifecycle(c, name, in.Body.Lifecycle, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "data-streams-attach-ilm",
		Method:      http.MethodPut,
		Path:        "/data_streams/{name}/ilm",
		Summary:     "Attach ILM policy to data stream",
		Tags:        []string{"data-streams"},
	}, func(ctx context.Context, in *DataStreamAttachILMIn) (*Output[DataStreamAttachILMResult], error) {
		name := strings.TrimSpace(in.Name)
		policy := strings.TrimSpace(in.Body.Policy)
		if name == "" {
			return failMsg[DataStreamAttachILMResult](http.StatusBadRequest, "data stream name is required")
		}
		if policy == "" {
			return failMsg[DataStreamAttachILMResult](http.StatusBadRequest, "ILM policy is required")
		}
		t, err := clusterTarget(ctx)
		if err != nil {
			return failMsg[DataStreamAttachILMResult](http.StatusBadRequest, err.Error())
		}
		streams, err := d.Client.GetDataStreams(ctx, t)
		if err != nil {
			return failMsg[DataStreamAttachILMResult](http.StatusInternalServerError, err.Error())
		}
		if !streams.IsSuccess() {
			return fail[DataStreamAttachILMResult](streams.Status, streams.Body)
		}
		target, found := dataStreamTarget(streams.Body, name)
		if !found {
			return failMsg[DataStreamAttachILMResult](http.StatusNotFound, "data stream not found")
		}
		if target.Template == "" {
			return failMsg[DataStreamAttachILMResult](http.StatusBadRequest, "data stream has no composable index template")
		}
		template, err := d.Client.GetComposableIndexTemplate(ctx, target.Template, t)
		if err != nil {
			return failMsg[DataStreamAttachILMResult](http.StatusInternalServerError, err.Error())
		}
		if !template.IsSuccess() {
			return fail[DataStreamAttachILMResult](template.Status, template.Body)
		}
		updatedTemplate, err := composableTemplateWithILM(template.Body, policy)
		if err != nil {
			return failMsg[DataStreamAttachILMResult](http.StatusBadRequest, err.Error())
		}
		putTemplate, err := d.Client.PutComposableIndexTemplate(ctx, target.Template, updatedTemplate, t)
		if err != nil {
			return failMsg[DataStreamAttachILMResult](http.StatusInternalServerError, err.Error())
		}
		if !putTemplate.IsSuccess() {
			return fail[DataStreamAttachILMResult](putTemplate.Status, putTemplate.Body)
		}

		result := DataStreamAttachILMResult{
			DataStream:      name,
			Policy:          policy,
			Template:        target.Template,
			TemplateUpdated: true,
		}
		if in.Body.UpdateBackingIndices && len(target.BackingIndices) > 0 {
			putSettings, err := d.Client.PutIndexLifecycleSettings(ctx, target.BackingIndices, policy, t)
			if err != nil {
				return failMsg[DataStreamAttachILMResult](http.StatusInternalServerError, err.Error())
			}
			if !putSettings.IsSuccess() {
				return fail[DataStreamAttachILMResult](putSettings.Status, putSettings.Body)
			}
			result.BackingIndicesUpdated = true
		}
		if in.Body.Rollover {
			rollover, err := d.Client.RolloverDataStream(ctx, name, t)
			if err != nil {
				return failMsg[DataStreamAttachILMResult](http.StatusInternalServerError, err.Error())
			}
			if !rollover.IsSuccess() {
				return fail[DataStreamAttachILMResult](rollover.Status, rollover.Body)
			}
			result.RolledOver = true
		}
		return ok(http.StatusOK, result)
	})

	huma.Register(api, huma.Operation{
		OperationID: "data-streams-detach-ilm",
		Method:      http.MethodDelete,
		Path:        "/data_streams/{name}/ilm",
		Summary:     "Detach ILM policy from data stream",
		Tags:        []string{"data-streams"},
	}, func(ctx context.Context, in *DataStreamDetachILMIn) (*Output[DataStreamDetachILMResult], error) {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return failMsg[DataStreamDetachILMResult](http.StatusBadRequest, "data stream name is required")
		}
		t, err := clusterTarget(ctx)
		if err != nil {
			return failMsg[DataStreamDetachILMResult](http.StatusBadRequest, err.Error())
		}
		streams, err := d.Client.GetDataStreams(ctx, t)
		if err != nil {
			return failMsg[DataStreamDetachILMResult](http.StatusInternalServerError, err.Error())
		}
		if !streams.IsSuccess() {
			return fail[DataStreamDetachILMResult](streams.Status, streams.Body)
		}
		target, found := dataStreamTarget(streams.Body, name)
		if !found {
			return failMsg[DataStreamDetachILMResult](http.StatusNotFound, "data stream not found")
		}
		if target.Template == "" {
			return failMsg[DataStreamDetachILMResult](http.StatusBadRequest, "data stream has no composable index template")
		}
		template, err := d.Client.GetComposableIndexTemplate(ctx, target.Template, t)
		if err != nil {
			return failMsg[DataStreamDetachILMResult](http.StatusInternalServerError, err.Error())
		}
		if !template.IsSuccess() {
			return fail[DataStreamDetachILMResult](template.Status, template.Body)
		}
		updatedTemplate, err := composableTemplateWithoutILM(template.Body)
		if err != nil {
			return failMsg[DataStreamDetachILMResult](http.StatusBadRequest, err.Error())
		}
		putTemplate, err := d.Client.PutComposableIndexTemplate(ctx, target.Template, updatedTemplate, t)
		if err != nil {
			return failMsg[DataStreamDetachILMResult](http.StatusInternalServerError, err.Error())
		}
		if !putTemplate.IsSuccess() {
			return fail[DataStreamDetachILMResult](putTemplate.Status, putTemplate.Body)
		}

		result := DataStreamDetachILMResult{
			DataStream:      name,
			Template:        target.Template,
			TemplateUpdated: true,
		}
		if in.UpdateBackingIndices && len(target.BackingIndices) > 0 {
			putSettings, err := d.Client.ClearIndexLifecycleSettings(ctx, target.BackingIndices, t)
			if err != nil {
				return failMsg[DataStreamDetachILMResult](http.StatusInternalServerError, err.Error())
			}
			if !putSettings.IsSuccess() {
				return fail[DataStreamDetachILMResult](putSettings.Status, putSettings.Body)
			}
			result.BackingIndicesUpdated = true
		}
		return ok(http.StatusOK, result)
	})
}

func successfulBody(response elastic.Response) json.RawMessage {
	if response.IsSuccess() {
		return response.Body
	}
	return nil
}

type dataStreamTargetInfo struct {
	Template       string
	BackingIndices []string
}

func dataStreamTarget(raw json.RawMessage, name string) (dataStreamTargetInfo, bool) {
	var root struct {
		DataStreams []struct {
			Name     string `json:"name"`
			Template string `json:"template"`
			Indices  []struct {
				IndexName string `json:"index_name"`
			} `json:"indices"`
		} `json:"data_streams"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return dataStreamTargetInfo{}, false
	}
	for _, stream := range root.DataStreams {
		if stream.Name != name {
			continue
		}
		indices := make([]string, 0, len(stream.Indices))
		for _, index := range stream.Indices {
			if index.IndexName != "" {
				indices = append(indices, index.IndexName)
			}
		}
		return dataStreamTargetInfo{Template: stream.Template, BackingIndices: indices}, true
	}
	return dataStreamTargetInfo{}, false
}

func composableTemplateWithILM(raw json.RawMessage, policy string) (json.RawMessage, error) {
	var root struct {
		IndexTemplates []struct {
			IndexTemplate map[string]any `json:"index_template"`
		} `json:"index_templates"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	if len(root.IndexTemplates) == 0 || root.IndexTemplates[0].IndexTemplate == nil {
		return nil, errors.New("composable index template not found")
	}
	indexTemplate := root.IndexTemplates[0].IndexTemplate
	template, _ := indexTemplate["template"].(map[string]any)
	if template == nil {
		template = map[string]any{}
		indexTemplate["template"] = template
	}
	settings, _ := template["settings"].(map[string]any)
	if settings == nil {
		settings = map[string]any{}
		template["settings"] = settings
	}
	settings["index.lifecycle.name"] = policy
	return json.Marshal(indexTemplate)
}

func composableTemplateWithoutILM(raw json.RawMessage) (json.RawMessage, error) {
	var root struct {
		IndexTemplates []struct {
			IndexTemplate map[string]any `json:"index_template"`
		} `json:"index_templates"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	if len(root.IndexTemplates) == 0 || root.IndexTemplates[0].IndexTemplate == nil {
		return nil, errors.New("composable index template not found")
	}
	indexTemplate := root.IndexTemplates[0].IndexTemplate
	template, _ := indexTemplate["template"].(map[string]any)
	settings, _ := template["settings"].(map[string]any)
	deleteILMSetting(settings)
	return json.Marshal(indexTemplate)
}

func deleteILMSetting(settings map[string]any) {
	if settings == nil {
		return
	}
	delete(settings, "index.lifecycle.name")
	index, _ := settings["index"].(map[string]any)
	lifecycle, _ := index["lifecycle"].(map[string]any)
	delete(lifecycle, "name")
	if len(lifecycle) == 0 && index != nil {
		delete(index, "lifecycle")
	}
	if len(index) == 0 {
		delete(settings, "index")
	}
}
