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
    trusted_proxies:
      - "10.0.0.10/32"

server:
  secret: "${APPLICATION_SECRET}"
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

`groups_header` can contain multiple groups separated by `group_separator`.

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
- Do not trust identity headers from arbitrary clients.
- Strip incoming identity headers at the edge before oauth2-proxy or your reverse proxy sets them.

