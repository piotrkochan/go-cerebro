# Microsoft Entra ID Authentication

Go Cerebro can authenticate users directly with Microsoft Entra ID using OIDC authorization-code flow.

## Entra ID App Registration

Create an app registration in Microsoft Entra ID and configure a web redirect URI:

```text
https://cerebro.example.org/auth/entraid/callback
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
    redirect_url: "https://cerebro.example.org/auth/entraid/callback"
    username_claim: "preferred_username"
    groups_claim: "groups"
    scopes: ["openid", "profile", "email"]

server:
  secret: "${APPLICATION_SECRET}"
  cookie_secure: true
  csrf_enabled: true
```

If `redirect_url` is omitted, Cerebro derives it from the incoming request and forwarded host/proto headers. Set it explicitly when Cerebro is behind a reverse proxy and the external URL differs from the internal URL.

## RBAC Groups

Cerebro reads groups from the configured ID token claim. It does not call Microsoft Graph.

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

## Security Notes

- Keep `client_secret` only in backend config or secret management.
- Serve Cerebro over HTTPS.
- Keep `server.cookie_secure: true` for browser-facing deployments.
- Register only trusted redirect URIs in Entra ID.
- Avoid relying on Microsoft Graph group overage for now; Cerebro validates token claims only.

