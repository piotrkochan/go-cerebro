# Generic OAuth / OIDC Authentication

Go Cerebro can authenticate users with a generic OAuth2 or OpenID Connect provider.

Prefer OIDC when the provider supports it. OIDC validates the ID token, audience and nonce. Generic OAuth2 mode exchanges the code and then reads identity from `userinfo_url`.

## OIDC Discovery

```yaml
auth:
  oauth:
    enabled: true
    name: "Dex"
    issuer_url: "https://dex.example.org"
    client_id: "${OAUTH_CLIENT_ID}"
    client_secret: "${OAUTH_CLIENT_SECRET}"
    redirect_url: "https://cerebro.example.org/auth/oauth/oauth/callback"
    username_claim: "preferred_username"
    groups_claim: "groups"
    default_groups: ["cerebro-viewers"]
    scopes: ["openid", "profile", "email"]

server:
  public_url: "https://cerebro.example.org"
  secret: "${APPLICATION_SECRET}"
  cookie_secure: true
  csrf_enabled: true
```

## OAuth2 Userinfo

```yaml
auth:
  oauth:
    enabled: true
    name: "GitHub"
    auth_url: "https://github.com/login/oauth/authorize"
    token_url: "https://github.com/login/oauth/access_token"
    userinfo_url: "https://api.github.com/user"
    client_id: "${OAUTH_CLIENT_ID}"
    client_secret: "${OAUTH_CLIENT_SECRET}"
    redirect_url: "https://cerebro.example.org/auth/oauth/oauth/callback"
    username_claim: "login"
    groups_claim: "teams"
    scopes: ["read:user"]
```

## RBAC Groups

Cerebro adds `auth.oauth.default_groups` and reads additional groups from `auth.oauth.groups_claim`.
Bind them in RBAC as `group:<value>`:

```yaml
rbac:
  enabled: true
  bindings:
    - {subject: "group:cerebro-admins", role: "role:admin"}
  policies:
    - {subject: "role:admin", resource: "*", action: "*", object: "*", effect: "allow"}
```

## Security Notes

- Keep `client_secret` only in backend config or secret management.
- Use HTTPS provider endpoints. HTTP is accepted only for localhost/loopback tests.
- Register the exact provider callback URL, for example `https://cerebro.example.org/auth/oauth/github/callback`, or your configured `redirect_url`.
- Prefer OIDC over plain OAuth2 when possible.
