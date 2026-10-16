# Trusted Proxy Authentication

Go Cerebro can trust authentication performed by a reverse proxy such as oauth2-proxy, Authelia, Traefik ForwardAuth or nginx `auth_request`.

In this mode the proxy authenticates the user and passes identity headers to Cerebro. Cerebro accepts those headers only when the request comes from a configured trusted proxy IP/CIDR.

## Config

```yaml
auth:
  proxy:
    enabled: true
    user_header: "X-Forwarded-User"
    groups_header: "X-Forwarded-Groups"
    group_separator: ","
    default_groups: ["cerebro-viewers"]
    logout_url: "/oauth2/sign_out?rd=/oauth2/sign_in"
    trusted_proxies:
      - "10.0.0.10/32"

server:
  secret: "${APPLICATION_SECRET}"
  public_url: "https://cerebro.example.org"
  trusted_proxies:
    - "10.0.0.10/32"
```

## oauth2-proxy Example

Configure oauth2-proxy to set a user header and group/email/domain headers appropriate for your provider. For GitHub, teams usually require oauth2-proxy provider settings and GitHub organization/team configuration on the proxy side.

Cerebro only needs the final trusted headers:

```yaml
auth:
  proxy:
    enabled: true
    user_header: "X-Forwarded-User"
    groups_header: "X-Forwarded-Groups"
    trusted_proxies: ["127.0.0.1/32"]
```

## RBAC Groups

`default_groups` are added to every user from this provider. `groups_header` can contain multiple
additional groups separated by `group_separator`.

`logout_url` is optional. When set, `/auth/logout` returns this URL to JSON clients so the frontend
can complete sign-out at the trusted proxy without knowing which auth provider was used.

```http
X-Forwarded-User: alice@example.org
X-Forwarded-Groups: cerebro-admins,operators
```

RBAC bindings then use the `group:` prefix:

```yaml
rbac:
  enabled: true
  bindings:
    - {subject: "group:cerebro-admins", role: "role:admin"}
```

## Security Notes

- Never expose Cerebro directly when `auth.proxy.enabled: true`.
- Restrict network access so only the trusted proxy can reach Cerebro.
- Set `trusted_proxies` to exact proxy IPs/CIDRs.
- Use `server.public_url` or `server.trusted_proxies` when the proxy terminates TLS or rewrites the external host.
- Do not trust identity headers from arbitrary clients.
- Strip incoming identity headers at the edge before oauth2-proxy or your reverse proxy sets them.
