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
	ClusterPath
}

type AnalysisIndexIn struct {
	ClusterPath
	Index string `path:"index" doc:"Index name."`
}

type AnalysisAnalyzerIn struct {
	ClusterPath
	Index string `path:"index" doc:"Index name."`
	Body  struct {
		Analyzer string `json:"analyzer" required:"true" doc:"Analyzer name."`
		Text     string `json:"text" required:"true" doc:"Text to analyze."`
	}
}

type AnalysisFieldIn struct {
	ClusterPath
	Index string `path:"index" doc:"Index name."`
	Body  struct {
		Field string `json:"field" required:"true" doc:"Field name."`
		Text  string `json:"text" required:"true" doc:"Text to analyze."`
	}
}

func (d *Deps) RegisterAnalysis(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "analysis-indices",
		Method:      http.MethodGet,
		Path:        "/analysis/indices",
		Summary:     "List open indices",
		Description: "Returns names of open indices available for analysis.",
		Tags:        []string{"analysis"},
	}, func(ctx context.Context, in *AnalysisHostIn) (*Output[List[string]], error) {
		return transformListResp(ctx, d,
			func(c context.Context, t elastic.Server) (elastic.Response, error) { return d.Client.GetIndices(c, t) },
			transform.OpenIndices)
	})

	huma.Register(api, huma.Operation{
		OperationID: "analysis-analyzers",
		Method:      http.MethodGet,
		Path:        "/analysis/indices/{index}/analyzers",
		Summary:     "List index analyzers",
		Description: "Returns analyzer names defined in the index settings.",
		Tags:        []string{"analysis"},
	}, func(ctx context.Context, in *AnalysisIndexIn) (*Output[List[string]], error) {
		return transformListResp(ctx, d,
			func(c context.Context, t elastic.Server) (elastic.Response, error) {
				return d.Client.GetIndexSettings(c, in.Index, t)
			},
			func(raw json.RawMessage) []string { return transform.IndexAnalyzers(in.Index, raw) })
	})

	huma.Register(api, huma.Operation{
		OperationID: "analysis-fields",
		Method:      http.MethodGet,
		Path:        "/analysis/indices/{index}/fields",
		Summary:     "List analyzable fields",
		Description: "Returns text fields of the index that can be analyzed.",
		Tags:        []string{"analysis"},
	}, func(ctx context.Context, in *AnalysisIndexIn) (*Output[List[string]], error) {
		return transformListResp(ctx, d,
			func(c context.Context, t elastic.Server) (elastic.Response, error) {
				return d.Client.GetIndexMapping(c, in.Index, t)
			},
			func(raw json.RawMessage) []string { return transform.IndexFields(in.Index, raw) })
	})

	huma.Register(api, huma.Operation{
		OperationID: "analysis-analyze-analyzer",
		Method:      http.MethodPost,
		Path:        "/analysis/indices/{index}/analyzers/_analyze",
		Summary:     "Analyze text by analyzer",
		Description: "Runs _analyze with the given analyzer and returns the produced tokens.",
		Tags:        []string{"analysis"},
	}, func(ctx context.Context, in *AnalysisAnalyzerIn) (*RawOutput, error) {
		return transformRawResp(ctx, d, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.AnalyzeTextByAnalyzer(c, in.Index, in.Body.Analyzer, in.Body.Text, t)
		}, transform.Tokens)
	})

	huma.Register(api, huma.Operation{
		OperationID: "analysis-analyze-field",
		Method:      http.MethodPost,
		Path:        "/analysis/indices/{index}/fields/_analyze",
		Summary:     "Analyze text by field",
		Description: "Runs _analyze against the analyzer of the given field and returns the produced tokens.",
		Tags:        []string{"analysis"},
	}, func(ctx context.Context, in *AnalysisFieldIn) (*RawOutput, error) {
		return transformRawResp(ctx, d, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.AnalyzeTextByField(c, in.Index, in.Body.Field, in.Body.Text, t)
		}, transform.Tokens)
	})
}
