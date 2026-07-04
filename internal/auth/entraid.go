package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/lmenezes/cerebro/internal/config"
	"golang.org/x/oauth2"
)

const (
	defaultEntraIDUsernameClaim = "preferred_username"
	defaultEntraIDGroupsClaim   = "groups"
)

type EntraIDProvider struct {
	tenantID      string
	issuerURL     string
	clientID      string
	clientSecret  string
	redirectURL   string
	scopes        []string
	usernameClaim string
	groupsClaim   string

	mu       sync.Mutex
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
}

func NewEntraIDProvider(settings config.EntraIDAuth) (*EntraIDProvider, error) {
	tenantID := strings.TrimSpace(settings.TenantID)
	issuerURL := strings.TrimSpace(settings.IssuerURL)
	clientID := strings.TrimSpace(settings.ClientID)
	if tenantID == "" && issuerURL == "" || clientID == "" || settings.ClientSecret == "" {
		return nil, errors.New("entra id auth requires tenant_id, client_id and client_secret")
	}
	if strings.Contains(tenantID, "/") {
		return nil, errors.New("entra id tenant_id must not contain slashes")
	}
	scopes := append([]string(nil), settings.Scopes...)
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}
	usernameClaim := strings.TrimSpace(settings.UsernameClaim)
	if usernameClaim == "" {
		usernameClaim = defaultEntraIDUsernameClaim
	}
	groupsClaim := strings.TrimSpace(settings.GroupsClaim)
	if groupsClaim == "" {
		groupsClaim = defaultEntraIDGroupsClaim
	}
	return &EntraIDProvider{
		tenantID:      tenantID,
		issuerURL:     issuerURL,
		clientID:      clientID,
		clientSecret:  settings.ClientSecret,
		redirectURL:   strings.TrimSpace(settings.RedirectURL),
		scopes:        scopes,
		usernameClaim: usernameClaim,
		groupsClaim:   groupsClaim,
	}, nil
}

func (p *EntraIDProvider) ConfiguredRedirectURL() string {
	if p == nil {
		return ""
	}
	return p.redirectURL
}

func (p *EntraIDProvider) AuthCodeURL(ctx context.Context, redirectURL, state, nonce string) (string, error) {
	oauthConfig, err := p.oauthConfig(ctx, redirectURL)
	if err != nil {
		return "", err
	}
	return oauthConfig.AuthCodeURL(state, oidc.Nonce(nonce)), nil
}

func (p *EntraIDProvider) Exchange(ctx context.Context, redirectURL, code, nonce string) (Identity, error) {
	oauthConfig, err := p.oauthConfig(ctx, redirectURL)
	if err != nil {
		return Identity{}, err
	}
	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		return Identity{}, fmt.Errorf("exchange entra id auth code: %w", err)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return Identity{}, errors.New("entra id token response did not include id_token")
	}
	verifier, err := p.idTokenVerifier(ctx)
	if err != nil {
		return Identity{}, err
	}
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return Identity{}, fmt.Errorf("verify entra id token: %w", err)
	}
	if !subtleConstantTimeEqual(idToken.Nonce, nonce) {
		return Identity{}, errors.New("invalid entra id nonce")
	}
	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return Identity{}, fmt.Errorf("decode entra id claims: %w", err)
	}
	username := firstClaimString(claims, p.usernameClaim, "preferred_username", "email", "name", "sub")
	if username == "" {
		username = idToken.Subject
	}
	if entraIDGroupsOverage(claims, p.groupsClaim) {
		return Identity{}, errors.New("entra id groups overage is not supported; configure app roles or a smaller groups claim")
	}
	return Identity{
		Username: username,
		Groups:   claimStringSlice(claims[p.groupsClaim]),
		Provider: "entra_id",
	}, nil
}

func (p *EntraIDProvider) oauthConfig(ctx context.Context, redirectURL string) (oauth2.Config, error) {
	provider, err := p.oidcProvider(ctx)
	if err != nil {
		return oauth2.Config{}, err
	}
	return oauth2.Config{
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  redirectURL,
		Scopes:       append([]string(nil), p.scopes...),
	}, nil
}

func (p *EntraIDProvider) idTokenVerifier(ctx context.Context) (*oidc.IDTokenVerifier, error) {
	if _, err := p.oidcProvider(ctx); err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.verifier, nil
}

func (p *EntraIDProvider) oidcProvider(ctx context.Context) (*oidc.Provider, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.provider != nil {
		return p.provider, nil
	}
	issuer := p.issuerURL
	if issuer == "" {
		issuer = "https://login.microsoftonline.com/" + url.PathEscape(p.tenantID) + "/v2.0"
	}
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("discover entra id provider: %w", err)
	}
	p.provider = provider
	p.verifier = provider.Verifier(&oidc.Config{ClientID: p.clientID})
	return provider, nil
}

func firstClaimString(claims map[string]any, names ...string) string {
	for _, name := range names {
		if value, ok := claims[name].(string); ok && value != "" {
			return value
		}
	}
	return ""
}

func claimStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if typed == "" {
			return nil
		}
		return []string{typed}
	default:
		return nil
	}
}

func entraIDGroupsOverage(claims map[string]any, groupsClaim string) bool {
	if _, ok := claims[groupsClaim]; ok {
		return false
	}
	names, ok := claims["_claim_names"].(map[string]any)
	if !ok {
		return false
	}
	_, ok = names[groupsClaim]
	return ok
}
