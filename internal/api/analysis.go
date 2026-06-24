package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/elastic"
	"github.com/lmenezes/cerebro/internal/transform"
)

type AnalysisHostIn struct {
	Body HostBody
}

type AnalysisIndexIn struct {
	Body struct {
		HostBody
		Index string `json:"index" required:"true" doc:"Index name."`
	}
}

type AnalysisAnalyzerIn struct {
	Body struct {
		HostBody
		Index    string `json:"index" required:"true" doc:"Index name."`
		Analyzer string `json:"analyzer" required:"true" doc:"Analyzer name."`
		Text     string `json:"text" required:"true" doc:"Text to analyze."`
	}
}

type AnalysisFieldIn struct {
	Body struct {
		HostBody
		Index string `json:"index" required:"true" doc:"Index name."`
		Field string `json:"field" required:"true" doc:"Field name."`
		Text  string `json:"text" required:"true" doc:"Text to analyze."`
	}
}

func (d *Deps) RegisterAnalysis(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "analysis-indices",
		Method:      http.MethodPost,
		Path:        "/analysis/indices",
		Summary:     "List open indices",
		Description: "Returns names of open indices available for analysis.",
		Tags:        []string{"analysis"},
	}, func(ctx context.Context, in *AnalysisHostIn) (*Output[List[string]], error) {
		return transformListResp(ctx, d, in.Body,
			func(c context.Context, t elastic.Server) (elastic.Response, error) { return d.Client.GetIndices(c, t) },
			transform.OpenIndices)
	})

	huma.Register(api, huma.Operation{
		OperationID: "analysis-analyzers",
		Method:      http.MethodPost,
		Path:        "/analysis/analyzers",
		Summary:     "List index analyzers",
		Description: "Returns analyzer names defined in the index settings.",
		Tags:        []string{"analysis"},
	}, func(ctx context.Context, in *AnalysisIndexIn) (*Output[List[string]], error) {
		return transformListResp(ctx, d, in.Body.HostBody,
			func(c context.Context, t elastic.Server) (elastic.Response, error) {
				return d.Client.GetIndexSettings(c, in.Body.Index, t)
			},
			func(raw json.RawMessage) []string { return transform.IndexAnalyzers(in.Body.Index, raw) })
	})

	huma.Register(api, huma.Operation{
		OperationID: "analysis-fields",
		Method:      http.MethodPost,
		Path:        "/analysis/fields",
		Summary:     "List analyzable fields",
		Description: "Returns text fields of the index that can be analyzed.",
		Tags:        []string{"analysis"},
	}, func(ctx context.Context, in *AnalysisIndexIn) (*Output[List[string]], error) {
		return transformListResp(ctx, d, in.Body.HostBody,
			func(c context.Context, t elastic.Server) (elastic.Response, error) {
				return d.Client.GetIndexMapping(c, in.Body.Index, t)
			},
			func(raw json.RawMessage) []string { return transform.IndexFields(in.Body.Index, raw) })
	})

	huma.Register(api, huma.Operation{
		OperationID: "analysis-analyze-analyzer",
		Method:      http.MethodPost,
		Path:        "/analysis/analyze/analyzer",
		Summary:     "Analyze text by analyzer",
		Description: "Runs _analyze with the given analyzer and returns the produced tokens.",
		Tags:        []string{"analysis"},
	}, func(ctx context.Context, in *AnalysisAnalyzerIn) (*RawOutput, error) {
		return transformRawResp(ctx, d, in.Body.HostBody, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.AnalyzeTextByAnalyzer(c, in.Body.Index, in.Body.Analyzer, in.Body.Text, t)
		}, transform.Tokens)
	})

	huma.Register(api, huma.Operation{
		OperationID: "analysis-analyze-field",
		Method:      http.MethodPost,
		Path:        "/analysis/analyze/field",
		Summary:     "Analyze text by field",
		Description: "Runs _analyze against the analyzer of the given field and returns the produced tokens.",
		Tags:        []string{"analysis"},
	}, func(ctx context.Context, in *AnalysisFieldIn) (*RawOutput, error) {
		return transformRawResp(ctx, d, in.Body.HostBody, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.AnalyzeTextByField(c, in.Body.Index, in.Body.Field, in.Body.Text, t)
		}, transform.Tokens)
	})
}
