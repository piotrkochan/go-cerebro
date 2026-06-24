package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/auth"
	"github.com/lmenezes/cerebro/internal/transform"
)

type RestIndexIn struct {
	Body HostBody
}

type RestRequestIn struct {
	Body struct {
		HostBody
		Method string          `json:"method" required:"true"`
		Path   string          `json:"path" required:"true"`
		Data   json.RawMessage `json:"data,omitempty"`
	}
}

type RestHistoryIn struct {
	Body HostBody
}

type RestRequestResponse struct {
	Status int             `json:"status" doc:"HTTP status returned by Elasticsearch."`
	Data   json.RawMessage `json:"data" doc:"Raw Elasticsearch response payload."`
}

func (d *Deps) RegisterRest(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "rest-index",
		Method:      http.MethodPost,
		Path:        "/rest",
		Tags:        []string{"rest"},
	}, func(ctx context.Context, in *RestIndexIn) (*RawOutput, error) {
		t, err := d.resolveTarget(httpRequest(ctx), in.Body)
		if err != nil {
			return failMsg[RawResponse](400, err.Error())
		}
		resp, err := d.Client.GetAliases(ctx, t)
		if err != nil {
			return failMsg[RawResponse](500, err.Error())
		}
		if !resp.IsSuccess() {
			return raw(resp.Status, resp.Body)
		}
		out := map[string]any{
			"indices": transform.AutocompletionIndices(resp.Body),
			"host":    t.Host.Host,
		}
		body, _ := json.Marshal(out)
		return raw(resp.Status, body)
	})

	huma.Register(api, huma.Operation{
		OperationID: "rest-request",
		Method:      http.MethodPost,
		Path:        "/rest/request",
		Tags:        []string{"rest"},
	}, func(ctx context.Context, in *RestRequestIn) (*Output[RestRequestResponse], error) {
		t, err := d.resolveTarget(httpRequest(ctx), in.Body.HostBody)
		if err != nil {
			return failMsg[RestRequestResponse](400, err.Error())
		}
		resp, err := d.Client.ExecuteRequest(ctx, in.Body.Method, in.Body.Path, in.Body.Data, t)
		if err != nil {
			return failMsg[RestRequestResponse](500, err.Error())
		}
		if d.History != nil {
			bodyStr := "{}"
			if len(in.Body.Data) > 0 {
				var asString string
				if err := json.Unmarshal(in.Body.Data, &asString); err == nil {
					bodyStr = asString
				} else {
					bodyStr = string(in.Body.Data)
				}
			}
			user := auth.UserFrom(ctx)
			if err := d.History.Save(ctx, user, in.Body.Path, in.Body.Method, bodyStr); err != nil {
				slog.Error("save rest history", "err", err)
			}
		}
		return ok(200, RestRequestResponse{Status: resp.Status, Data: resp.Body})
	})

	huma.Register(api, huma.Operation{
		OperationID: "rest-history",
		Method:      http.MethodPost,
		Path:        "/rest/history",
		Tags:        []string{"rest"},
	}, func(ctx context.Context, _ *RestHistoryIn) (*RawOutput, error) {
		if d.History == nil {
			return raw(200, json.RawMessage(`[]`))
		}
		user := auth.UserFrom(ctx)
		rows, err := d.History.All(ctx, user)
		if err != nil {
			return failMsg[RawResponse](500, err.Error())
		}
		out := make([]map[string]any, 0, len(rows))
		for _, r := range rows {
			out = append(out, map[string]any{
				"path":       r.Path,
				"method":     r.Method,
				"body":       r.Body,
				"created_at": time.UnixMilli(r.CreatedAt).UTC().Format("02/01 15:04:05"),
			})
		}
		body, _ := json.Marshal(out)
		return raw(200, body)
	})
}
