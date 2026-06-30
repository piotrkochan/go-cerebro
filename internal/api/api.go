package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/auth"
	"github.com/lmenezes/cerebro/internal/config"
	"github.com/lmenezes/cerebro/internal/elastic"
	"github.com/lmenezes/cerebro/internal/history"
)

type Deps struct {
	Cfg     *config.Config
	Client  elastic.Client
	History *history.Store
	Auth    *auth.Module
}

// Output is the Huma output wrapper for a typed response body.
type Output[T any] struct {
	Status int
	Body   T
}

// List is a typed collection response. Returning a top-level object lets Huma
// attach the response JSON Schema via `$schema`; top-level arrays cannot carry
// that property.
type List[T any] struct {
	Items []T `json:"items" doc:"Collection items."`
}

// RawResponse wraps arbitrary Elasticsearch JSON so every endpoint still has a
// schema-bearing response object. The raw upstream payload is exposed as `data`.
type RawResponse struct {
	Data json.RawMessage `json:"data" doc:"Raw Elasticsearch response payload."`
}

// RawOutput is the Huma output wrapper for endpoints that proxy Elasticsearch
// responses.
type RawOutput = Output[RawResponse]

// ok returns a typed payload with the upstream HTTP status preserved.
func ok[T any](status int, v T) (*Output[T], error) {
	return &Output[T]{Status: status, Body: v}, nil
}

func okList[T any](status int, v []T) (*Output[List[T]], error) {
	if v == nil {
		v = []T{}
	}
	return ok(status, List[T]{Items: v})
}

// fail turns an upstream error response into a standard Huma error response.
func fail[T any](status int, raw json.RawMessage) (*Output[T], error) {
	if status < http.StatusBadRequest {
		status = http.StatusInternalServerError
	}
	return nil, huma.NewError(status, string(raw))
}

// failMsg returns a standard Huma error response for Cerebro-side failures.
func failMsg[T any](status int, msg string) (*Output[T], error) {
	if status < http.StatusBadRequest {
		status = http.StatusInternalServerError
	}
	return nil, huma.NewError(status, msg)
}

// raw returns an Elasticsearch response body verbatim with the upstream HTTP
// status preserved.
func raw(status int, body json.RawMessage) (*RawOutput, error) {
	if body == nil {
		body = json.RawMessage("null")
	}
	return ok(status, RawResponse{Data: body})
}

// HostBody is embedded by request bodies that need to select a cluster target.
// Elasticsearch credentials are intentionally resolved only from backend config.
type HostBody struct {
	Host string `json:"host" required:"true" doc:"Name of the target Elasticsearch host as configured."`
}

// ClusterPath selects the target cluster for REST-style endpoints.
type ClusterPath struct {
	Cluster string `path:"cluster" doc:"Configured Elasticsearch host slug."`
}

// resolveTarget converts the selected host into an elastic.Server and forwards whitelisted headers.
func (d *Deps) resolveTarget(r *http.Request, hb HostBody) (elastic.Server, error) {
	if hb.Host == "" {
		return elastic.Server{}, errors.New("missing required parameter host")
	}
	host, ok := d.Cfg.HostByName(hb.Host)
	if !ok {
		host, ok = d.Cfg.HostBySlug(hb.Host)
	}
	if !ok {
		if !d.Cfg.ES.AllowAdHocHosts {
			return elastic.Server{}, errors.New("unknown elasticsearch host; add it to configuration or enable es.allow_ad_hoc_hosts")
		}
		if err := validateAdHocHost(hb.Host); err != nil {
			return elastic.Server{}, err
		}
		host = config.Host{Name: hb.Host, Host: hb.Host}
	}
	return elasticServer(r, host), nil
}

func (d *Deps) resolveClusterTarget(r *http.Request, slug string) (elastic.Server, error) {
	if slug == "" {
		return elastic.Server{}, errors.New("missing required parameter cluster")
	}
	host, ok := d.Cfg.HostBySlug(slug)
	if !ok {
		return elastic.Server{}, errors.New("unknown elasticsearch cluster slug")
	}
	return elasticServer(r, host), nil
}

func elasticServer(r *http.Request, host config.Host) elastic.Server {
	headers := [][2]string{}
	for _, h := range host.HeadersWhitelist {
		if r != nil {
			if v := r.Header.Get(h); v != "" {
				headers = append(headers, [2]string{h, v})
			}
		}
	}
	return elastic.Server{Host: host, Headers: headers}
}

type clusterTargetKey struct{}

type clusterTargetResult struct {
	target elastic.Server
	err    error
}

// ClusterTargetMiddleware resolves the `{cluster}` path once for the whole request.
func (d *Deps) ClusterTargetMiddleware(ctx huma.Context, next func(huma.Context)) {
	result := clusterTargetResult{}
	result.target, result.err = d.resolveClusterTarget(httpRequest(ctx.Context()), ctx.Param("cluster"))
	next(huma.WithValue(ctx, clusterTargetKey{}, result))
}

func clusterTarget(ctx context.Context) (elastic.Server, error) {
	result, ok := ctx.Value(clusterTargetKey{}).(clusterTargetResult)
	if !ok {
		return elastic.Server{}, errors.New("missing cluster target")
	}
	return result.target, result.err
}

func validateAdHocHost(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("elasticsearch host must use http or https")
	}
	if u.Host == "" {
		return errors.New("elasticsearch host must include a host")
	}
	if u.User != nil {
		return errors.New("credentials in elasticsearch host URL are not allowed")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return errors.New("elasticsearch host URL must not include query or fragment")
	}
	if strings.TrimRight(u.Path, "/") != "" {
		return errors.New("elasticsearch host URL must not include a path")
	}
	return nil
}

// httpRequest extracts the underlying http.Request from a Huma context (for header forwarding).
func httpRequest(ctx context.Context) *http.Request {
	if r, ok := ctx.Value(httpReqKey{}).(*http.Request); ok {
		return r
	}
	return nil
}

type httpReqKey struct{}

// WithHTTPRequest is used by middleware to expose the *http.Request to handlers (so they can read whitelisted headers).
func WithHTTPRequest(ctx context.Context, r *http.Request) context.Context {
	return context.WithValue(ctx, httpReqKey{}, r)
}

// passthrough is the universal handler shape for proxy endpoints: resolve the target host,
// run the Elasticsearch call and return its status and body verbatim.
func (d *Deps) passthrough(ctx context.Context, fn func(ctx context.Context, t elastic.Server) (elastic.Response, error)) (*RawOutput, error) {
	t, err := clusterTarget(ctx)
	if err != nil {
		return failMsg[RawResponse](400, err.Error())
	}
	resp, err := fn(ctx, t)
	if err != nil {
		return failMsg[RawResponse](500, err.Error())
	}
	return raw(resp.Status, resp.Body)
}

// transformResp converts a successful ES response into a typed payload via tf; errors pass straight through.
func transformResp[T any](ctx context.Context, d *Deps, fn func(ctx context.Context, t elastic.Server) (elastic.Response, error), tf func(json.RawMessage) T) (*Output[T], error) {
	t, err := clusterTarget(ctx)
	if err != nil {
		return failMsg[T](400, err.Error())
	}
	resp, err := fn(ctx, t)
	if err != nil {
		return failMsg[T](500, err.Error())
	}
	if !resp.IsSuccess() {
		return fail[T](resp.Status, resp.Body)
	}
	return ok(resp.Status, tf(resp.Body))
}

func transformRawResp(ctx context.Context, d *Deps, fn func(ctx context.Context, t elastic.Server) (elastic.Response, error), tf func(json.RawMessage) json.RawMessage) (*RawOutput, error) {
	t, err := clusterTarget(ctx)
	if err != nil {
		return failMsg[RawResponse](400, err.Error())
	}
	resp, err := fn(ctx, t)
	if err != nil {
		return failMsg[RawResponse](500, err.Error())
	}
	if !resp.IsSuccess() {
		return fail[RawResponse](resp.Status, resp.Body)
	}
	return raw(resp.Status, tf(resp.Body))
}

func transformListResp[T any](ctx context.Context, d *Deps, fn func(ctx context.Context, t elastic.Server) (elastic.Response, error), tf func(json.RawMessage) []T) (*Output[List[T]], error) {
	t, err := clusterTarget(ctx)
	if err != nil {
		return failMsg[List[T]](400, err.Error())
	}
	resp, err := fn(ctx, t)
	if err != nil {
		return failMsg[List[T]](500, err.Error())
	}
	if !resp.IsSuccess() {
		return fail[List[T]](resp.Status, resp.Body)
	}
	return okList(resp.Status, tf(resp.Body))
}

// firstError returns the first non-success response, or nil if all are 2xx.
func firstError(resps []elastic.Response) *elastic.Response {
	for i := range resps {
		if !resps[i].IsSuccess() {
			return &resps[i]
		}
	}
	return nil
}
