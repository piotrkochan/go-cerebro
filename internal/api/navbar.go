package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/auth"
)

type NavbarIn struct {
	Body HostBody
}

func (d *Deps) RegisterNavbar(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "navbar",
		Method:      http.MethodPost,
		Path:        "/navbar",
		Summary:     "Get navbar data",
		Description: "Returns the cluster health document (verbatim from _cluster/health), extended with a \"username\" field when authentication is enabled.",
		Tags:        []string{"navbar"},
	}, func(ctx context.Context, in *NavbarIn) (*RawOutput, error) {
		t, err := d.resolveTarget(httpRequest(ctx), in.Body)
		if err != nil {
			return failMsg[RawResponse](400, err.Error())
		}
		resp, err := d.Client.ClusterHealth(ctx, t)
		if err != nil {
			return failMsg[RawResponse](500, err.Error())
		}
		if !resp.IsSuccess() {
			return fail[RawResponse](resp.Status, resp.Body)
		}
		version := json.RawMessage(nil)
		if versionResp, err := d.Client.Main(ctx, t); err == nil && versionResp.IsSuccess() {
			var info map[string]json.RawMessage
			if err := json.Unmarshal(versionResp.Body, &info); err == nil {
				version = info["version"]
			}
		}
		if user := auth.UserFrom(ctx); user != "" {
			var m map[string]json.RawMessage
			if err := json.Unmarshal(resp.Body, &m); err == nil {
				name, _ := json.Marshal(user)
				m["username"] = name
				features, _ := json.Marshal(map[string]bool{"data_explorer": d.Cfg.Features.DataExplorer})
				m["features"] = features
				if len(version) > 0 {
					m["version"] = version
				}
				newBody, _ := json.Marshal(m)
				return raw(resp.Status, newBody)
			}
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(resp.Body, &m); err == nil {
			features, _ := json.Marshal(map[string]bool{"data_explorer": d.Cfg.Features.DataExplorer})
			m["features"] = features
			if len(version) > 0 {
				m["version"] = version
			}
			newBody, _ := json.Marshal(m)
			return raw(resp.Status, newBody)
		}
		return raw(resp.Status, resp.Body)
	})
}
