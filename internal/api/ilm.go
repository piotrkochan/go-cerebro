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

type ILMHostIn struct {
	ClusterPath
}

type ILMPolicyNameIn struct {
	ClusterPath
	Name string `path:"name" doc:"ILM policy name."`
}

type ILMPolicySaveIn struct {
	ClusterPath
	Name string `path:"name" doc:"ILM policy name."`
	Body struct {
		Policy json.RawMessage `json:"policy" required:"true" doc:"ILM policy body."`
	}
}

func (d *Deps) RegisterILM(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "ilm-policies-list",
		Method:      http.MethodGet,
		Path:        "/ilm/policies",
		Summary:     "List ILM policies",
		Tags:        []string{"ilm"},
	}, func(ctx context.Context, in *ILMHostIn) (*Output[List[transform.ILMPolicy]], error) {
		return transformListResp(ctx, d,
			func(c context.Context, t elastic.Server) (elastic.Response, error) {
				return d.Client.GetILMPolicies(c, t)
			},
			transform.ILMPolicies)
	})

	huma.Register(api, huma.Operation{
		OperationID: "ilm-policies-save",
		Method:      http.MethodPut,
		Path:        "/ilm/policies/{name}",
		Summary:     "Create or update ILM policy",
		Tags:        []string{"ilm"},
	}, func(ctx context.Context, in *ILMPolicySaveIn) (*RawOutput, error) {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return failMsg[RawResponse](http.StatusBadRequest, "ILM policy name is required")
		}
		if !json.Valid(in.Body.Policy) {
			return failMsg[RawResponse](http.StatusBadRequest, "policy must be valid JSON")
		}
		return d.passthrough(ctx, func(c context.Context, t elastic.Server) (elastic.Response, error) {
			return d.Client.PutILMPolicy(c, name, in.Body.Policy, t)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "ilm-policies-delete",
		Method:      http.MethodDelete,
		Path:        "/ilm/policies/{name}",
		Summary:     "Delete ILM policy",
		Tags:        []string{"ilm"},
	}, func(ctx context.Context, in *ILMPolicyNameIn) (*RawOutput, error) {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return failMsg[RawResponse](http.StatusBadRequest, "ILM policy name is required")
		}
		t, err := clusterTarget(ctx)
		if err != nil {
			return failMsg[RawResponse](http.StatusBadRequest, err.Error())
		}
		policies, err := d.Client.GetILMPolicies(ctx, t)
		if err != nil {
			return failMsg[RawResponse](http.StatusInternalServerError, err.Error())
		}
		if !policies.IsSuccess() {
			return fail[RawResponse](policies.Status, policies.Body)
		}
		for _, policy := range transform.ILMPolicies(policies.Body) {
			if policy.Name == name && ilmPolicyInUse(policy.InUseBy) {
				return failMsg[RawResponse](http.StatusConflict, "ILM policy is in use")
			}
		}
		resp, err := d.Client.DeleteILMPolicy(ctx, name, t)
		if err != nil {
			return failMsg[RawResponse](http.StatusInternalServerError, err.Error())
		}
		if !resp.IsSuccess() {
			return fail[RawResponse](resp.Status, resp.Body)
		}
		return raw(resp.Status, resp.Body)
	})
}

func ilmPolicyInUse(value transform.ILMPolicyInUseBy) bool {
	return len(value.Indices) > 0 || len(value.DataStreams) > 0 || len(value.ComposableTemplates) > 0
}
