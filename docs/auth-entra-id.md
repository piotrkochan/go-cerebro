# Microsoft Entra ID Authentication

Go Cerebro can authenticate users directly with Microsoft Entra ID using OIDC authorization-code flow.

## Entra ID App Registration

Create an app registration in Microsoft Entra ID and configure a web redirect URI:

```text
https://cerebro.example.org/auth/entraid/entra_id/callback
```

Create a client secret and store it outside source control.

## Cerebro Config

```yaml
auth:
  entra_id:
    enabled: true
    tenant_id: "${ENTRA_ID_TENANT_ID}"
    client_id: "${ENTRA_ID_CLIENT_ID}"
    client_secret: "${ENTRA_ID_CLIENT_SECRET}"
    redirect_url: "https://cerebro.example.org/auth/entraid/entra_id/callback"
    username_claim: "preferred_username"
    groups_claim: "groups"
    default_groups: ["cerebro-viewers"]
    scopes: ["openid", "profile", "email"]

server:
  public_url: "https://cerebro.example.org"
  trusted_proxies: ["10.0.0.10/32"]
  secret: "${APPLICATION_SECRET}"
  cookie_secure: true
  csrf_enabled: true
```

If `redirect_url` is omitted, Cerebro derives it from `server.public_url` or from the incoming request. Forwarded host/proto headers are used only when the request comes from `server.trusted_proxies`. Set `redirect_url` explicitly when the external callback URL differs from Cerebro's public URL plus `server.base_path`.

## RBAC Groups

Cerebro adds `auth.entra_id.default_groups` and reads additional groups from the configured ID token
claim. It does not call Microsoft Graph.

For RBAC group bindings, configure the Entra ID app registration to emit groups or app roles in the ID token, then bind them in RBAC:

```yaml
rbac:
  enabled: true
  bindings:
    - {subject: "group:cerebro-admins", role: "role:admin"}
  policies:
    - {subject: "role:admin", resource: "*", action: "*", object: "*", effect: "allow"}
```

If your tenant emits group object IDs instead of names, bind those IDs as `group:<id>` or use app roles and set `groups_claim` to the roles claim.

If Entra ID returns a group overage marker instead of the configured groups claim, Cerebro rejects the login. Configure app roles or reduce emitted groups so the ID token contains the claim Cerebro should use.

## Security Notes

- Keep `client_secret` only in backend config or secret management.
- Serve Cerebro over HTTPS.
- Keep `server.cookie_secure: true` for browser-facing deployments.
- Register only trusted redirect URIs in Entra ID.
- Do not rely on Microsoft Graph group overage; Cerebro validates token claims only.
