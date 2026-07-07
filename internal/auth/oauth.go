package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/lmenezes/cerebro/internal/config"
	"golang.org/x/oauth2"
)

const (
	defaultOAuthName          = "OAuth"
	defaultOAuthUsernameClaim = "preferred_username"
	defaultOAuthGroupsClaim   = "groups"
)

type OAuthProvider struct {
	name          string
	issuerURL     string
	authURL       string
	tokenURL      string
	userInfoURL   string
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

func NewOAuthProvider(settings config.OAuthAuth) (*OAuthProvider, error) {
	name := strings.TrimSpace(settings.Name)
	if name == "" {
		name = defaultOAuthName
	}
	clientID := strings.TrimSpace(settings.ClientID)
	if clientID == "" || settings.ClientSecret == "" {
		return nil, errors.New("oauth auth requires client_id and client_secret")
	}
	issuerURL := strings.TrimSpace(settings.IssuerURL)
	authURL := strings.TrimSpace(settings.AuthURL)
	tokenURL := strings.TrimSpace(settings.TokenURL)
	userInfoURL := strings.TrimSpace(settings.UserInfoURL)
	if issuerURL == "" && (authURL == "" || tokenURL == "" || userInfoURL == "") {
		return nil, errors.New("oauth auth requires issuer_url or auth_url, token_url and userinfo_url")
	}
	scopes := append([]string(nil), settings.Scopes...)
	if len(scopes) == 0 && issuerURL != "" {
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}
	usernameClaim := strings.TrimSpace(settings.UsernameClaim)
	if usernameClaim == "" {
		usernameClaim = defaultOAuthUsernameClaim
	}
	groupsClaim := strings.TrimSpace(settings.GroupsClaim)
	if groupsClaim == "" {
		groupsClaim = defaultOAuthGroupsClaim
	}
	return &OAuthProvider{
		name:          name,
		issuerURL:     issuerURL,
		authURL:       authURL,
		tokenURL:      tokenURL,
		userInfoURL:   userInfoURL,
		clientID:      clientID,
		clientSecret:  settings.ClientSecret,
		redirectURL:   strings.TrimSpace(settings.RedirectURL),
		scopes:        scopes,
		usernameClaim: usernameClaim,
		groupsClaim:   groupsClaim,
	}, nil
}

func (p *OAuthProvider) Name() string {
	if p == nil || p.name == "" {
		return defaultOAuthName
	}
	return p.name
}

func (p *OAuthProvider) ConfiguredRedirectURL() string {
	if p == nil {
		return ""
	}
	return p.redirectURL
}

func (p *OAuthProvider) AuthCodeURL(ctx context.Context, redirectURL, state, nonce string) (string, error) {
	oauthConfig, err := p.oauthConfig(ctx, redirectURL)
	if err != nil {
		return "", err
	}
	options := []oauth2.AuthCodeOption{oauth2.AccessTypeOnline}
	if p.issuerURL != "" {
		options = append(options, oidc.Nonce(nonce))
	}
	return oauthConfig.AuthCodeURL(state, options...), nil
}

func (p *OAuthProvider) Exchange(ctx context.Context, redirectURL, code, nonce string) (Identity, error) {
	oauthConfig, err := p.oauthConfig(ctx, redirectURL)
	if err != nil {
		return Identity{}, err
	}
	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		return Identity{}, fmt.Errorf("exchange oauth auth code: %w", err)
	}
	claims, subject, err := p.identityClaims(ctx, oauthConfig, token, nonce)
	if err != nil {
		return Identity{}, err
	}
	username := firstClaimString(claims, p.usernameClaim, "preferred_username", "login", "email", "name", "sub", "id")
	if username == "" {
		username = subject
	}
	if username == "" {
		return Identity{}, errors.New("oauth identity did not include a username claim")
	}
	return Identity{
		Username: username,
		Groups:   claimStringSlice(claims[p.groupsClaim]),
		Provider: "oauth",
	}, nil
}

func (p *OAuthProvider) identityClaims(ctx context.Context, oauthConfig oauth2.Config, token *oauth2.Token, nonce string) (map[string]any, string, error) {
	if p.issuerURL != "" {
		rawIDToken, ok := token.Extra("id_token").(string)
		if !ok || rawIDToken == "" {
			return nil, "", errors.New("oauth token response did not include id_token")
		}
		verifier, err := p.idTokenVerifier(ctx)
		if err != nil {
			return nil, "", err
		}
		idToken, err := verifier.Verify(ctx, rawIDToken)
		if err != nil {
			return nil, "", fmt.Errorf("verify oauth id token: %w", err)
		}
		if !subtleConstantTimeEqual(idToken.Nonce, nonce) {
			return nil, "", errors.New("invalid oauth nonce")
		}
		var claims map[string]any
		if err := idToken.Claims(&claims); err != nil {
			return nil, "", fmt.Errorf("decode oauth claims: %w", err)
		}
		return claims, idToken.Subject, nil
	}
	if token.AccessToken == "" {
		return nil, "", errors.New("oauth token response did not include access_token")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.userInfoURL, nil) // #nosec G107 -- userinfo_url is operator config validated by config.Load.
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	resp, err := oauthConfig.Client(ctx, token).Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fetch oauth userinfo: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, "", fmt.Errorf("fetch oauth userinfo: unexpected status %d", resp.StatusCode)
	}
	var claims map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return nil, "", fmt.Errorf("decode oauth userinfo: %w", err)
	}
	return claims, firstClaimString(claims, "sub", "id"), nil
}

func (p *OAuthProvider) oauthConfig(ctx context.Context, redirectURL string) (oauth2.Config, error) {
	endpoint := oauth2.Endpoint{
		AuthURL:  p.authURL,
		TokenURL: p.tokenURL,
	}
	if p.issuerURL != "" {
		provider, err := p.oidcProvider(ctx)
		if err != nil {
			return oauth2.Config{}, err
		}
		endpoint = provider.Endpoint()
	}
	return oauth2.Config{
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		Endpoint:     endpoint,
		RedirectURL:  redirectURL,
		Scopes:       append([]string(nil), p.scopes...),
	}, nil
}

func (p *OAuthProvider) idTokenVerifier(ctx context.Context) (*oidc.IDTokenVerifier, error) {
	if _, err := p.oidcProvider(ctx); err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.verifier, nil
}

func (p *OAuthProvider) oidcProvider(ctx context.Context) (*oidc.Provider, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.provider != nil {
		return p.provider, nil
	}
	provider, err := oidc.NewProvider(ctx, p.issuerURL)
	if err != nil {
		return nil, fmt.Errorf("discover oauth provider: %w", err)
	}
	p.provider = provider
	p.verifier = provider.Verifier(&oidc.Config{ClientID: p.clientID})
	return provider, nil
}
