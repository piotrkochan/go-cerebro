package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// RegisterAuth wires the JSON logout endpoint and browser-oriented login/callback handlers.
func (d *Deps) RegisterAuth(api huma.API, mux interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}) {
	huma.Register(api, huma.Operation{
		OperationID: "auth-logout",
		Method:      http.MethodPost,
		Path:        "/auth/logout",
		Summary:     "Log out",
		Description: "Clears the current Cerebro session and returns the URL the frontend should navigate to.",
		Tags:        []string{"auth"},
	}, func(ctx context.Context, _ *struct{}) (*LogoutOutput, error) {
		return d.logout(ctx)
	})

	mux.HandleFunc("GET /auth/status", d.handleAuthStatus)
	mux.HandleFunc("GET /auth/me", d.handleAuthMe)
	mux.HandleFunc("POST /auth/login", d.handleLogin)
	mux.HandleFunc("GET /auth/entraid/login", d.handleEntraIDLogin)
	mux.HandleFunc("GET /auth/entraid/callback", d.handleEntraIDCallback)
	mux.HandleFunc("GET /auth/entraid/{provider}/login", d.handleEntraIDLogin)
	mux.HandleFunc("GET /auth/entraid/{provider}/callback", d.handleEntraIDCallback)
	mux.HandleFunc("GET /auth/oauth/login", d.handleOAuthLogin)
	mux.HandleFunc("GET /auth/oauth/callback", d.handleOAuthCallback)
	mux.HandleFunc("GET /auth/oauth/{provider}/login", d.handleOAuthLogin)
	mux.HandleFunc("GET /auth/oauth/{provider}/callback", d.handleOAuthCallback)
}

type LogoutResponse struct {
	Schema      string `json:"$schema,omitempty" doc:"JSON schema URL for this response."`
	RedirectURL string `json:"redirect_url" doc:"URL the frontend should navigate to after logout."`
}

type LogoutOutput struct {
	Status    int
	SetCookie []string       `header:"Set-Cookie" hidden:"true"`
	Body      LogoutResponse `json:"body"`
}

func (d *Deps) logout(ctx context.Context) (*LogoutOutput, error) {
	r := httpRequest(ctx)
	if r == nil {
		return nil, huma.Error500InternalServerError("request context missing")
	}
	identity, _ := d.Auth.SessionIdentity(r)
	setCookies, err := d.Auth.ClearSessionCookies(r)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}
	d.auditAuth(r, "logout", identity, "success", "")
	return &LogoutOutput{
		Status:    http.StatusOK,
		SetCookie: setCookies,
		Body: LogoutResponse{
			Schema:      apiSchemaURL(d.Cfg.Server, r, "LogoutResponse"),
			RedirectURL: d.Auth.LogoutRedirectURL(identity, basePathFor(d, "/#/login")),
		},
	}, nil
}
