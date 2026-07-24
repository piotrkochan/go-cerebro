# Go Cerebro

<img align="right" width="220" src="./docs/assets/go-cerebro-logo-400x400.png" alt="Go Cerebro logo">

[![CI](https://github.com/piotrkochan/go-cerebro/actions/workflows/ci.yml/badge.svg)](https://github.com/piotrkochan/go-cerebro/actions/workflows/ci.yml)
[![Release](https://github.com/piotrkochan/go-cerebro/actions/workflows/release.yml/badge.svg)](https://github.com/piotrkochan/go-cerebro/actions/workflows/release.yml)
[![CodeQL](https://github.com/piotrkochan/go-cerebro/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/piotrkochan/go-cerebro/actions/workflows/github-code-scanning/codeql)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/piotrkochan/go-cerebro/badge)](https://scorecard.dev/viewer/?uri=github.com/piotrkochan/go-cerebro)
[![Go Report Card](https://goreportcard.com/badge/github.com/piotrkochan/go-cerebro)](https://goreportcard.com/report/github.com/piotrkochan/go-cerebro)
[![Latest Release](https://img.shields.io/github/v/release/piotrkochan/go-cerebro?logo=github)](https://github.com/piotrkochan/go-cerebro/releases/latest)
[![Docker Image](https://img.shields.io/badge/ghcr.io-go--cerebro-blue?logo=docker)](https://github.com/piotrkochan/go-cerebro/pkgs/container/go-cerebro)
[![License](https://img.shields.io/github/license/piotrkochan/go-cerebro)](./LICENSE)

Go Cerebro is a fork of the original [lmenezes/cerebro](https://github.com/lmenezes/cerebro) Elasticsearch web admin tool. The original application was Java + Angular; this fork is rewritten to Go, Huma, React and TypeScript while keeping the style, workflow and core functionality of the original Cerebro. It also adds new functionality that fits the same admin-console use case, including:

- backend and React frontend packaged into one binary for easy deployment
- data explorer for browsing, searching, inserting and editing index documents
- data streams and ILM management
- AWS OpenSearch SigV4 support with dedicated compatibility tests
- LDAPS, trusted proxy, Entra ID and generic OIDC authentication
- advanced RBAC

AI assistance was used during the refactor. Despite careful review, there may still be rough edges, so please keep that in mind while evaluating or using this version.

Go Cerebro is an independent project and is not affiliated with, endorsed by or sponsored by Elastic NV or any Elastic company. Elasticsearch, Elastic and related marks are trademarks of Elastic NV or its affiliates.

## Requirements

- Go Cerebro binary or Docker image
- configuration file
- reachable Elasticsearch or OpenSearch cluster

## Installation

Download a binary for your platform from [GitHub Releases](https://github.com/piotrkochan/go-cerebro/releases), make it executable and run it with a config file:

```sh
chmod +x cerebro-linux-amd64
./cerebro-linux-amd64 serve -config conf/application.yaml
```

You can also run the Docker image:

```sh
docker run --rm -p 9000:9000 \
  -v ./conf/application.yaml:/etc/cerebro/application.yaml:ro \
  ghcr.io/piotrkochan/go-cerebro:latest \
  serve -config /etc/cerebro/application.yaml
```

## Quick Start

Create a minimal `conf/application.yaml`:

```yaml
hosts:
  - id: "local-cluster"
    name: "Local cluster"
    host: "http://localhost:9200"

server:
  port: 9000
  secret: "local-dev-secret"
  cookie_secure: false

es:
  allow_ad_hoc_hosts: false

features:
  data_explorer: false
```

Then run Cerebro:

```sh
cerebro serve -config conf/application.yaml
```

Then open `http://localhost:9000`.

The development stack and live-reload setup are described in [CONTRIBUTING.md](./CONTRIBUTING.md).

For local development with Elasticsearch, use [conf/application.dev.yaml](./conf/application.dev.yaml).

## CLI

Serve the application:

```sh
cerebro serve -config conf/application.yaml
```

Generate the OpenAPI spec:

```sh
cerebro openapi -config conf/application.full-example.yaml > openapi/cerebro.json
```

Print the version:

```sh
cerebro version
```

## Configuration

Copy [conf/application.full-example.yaml](./conf/application.full-example.yaml) to `conf/application.yaml`, remove options you do not need and edit it for your environment.

Environment variables are expanded inside YAML string values. These direct overrides are also supported:

| Environment variable | Overrides |
| --- | --- |
| `CEREBRO_PORT` | `server.port` |
| `APPLICATION_SECRET` | `server.secret` |

| Option | Default | Description |
| --- | --- | --- |
| `hosts` | `[]` | Known Elasticsearch/OpenSearch clusters shown on the connect page. |
| `hosts[].id` | generated from `hosts[].name` | Stable cluster slug used in URLs and RBAC. Use lowercase letters, digits and hyphens. |
| `hosts[].name` | `hosts[].host` | Display name shown in the UI. |
| `hosts[].host` | required | Elasticsearch/OpenSearch HTTP URL. Credentials in the URL are rejected; use `hosts[].auth`. |
| `hosts[].auth.username` | empty | Username for this Elasticsearch/OpenSearch host. Kept server-side. |
| `hosts[].auth.password` | empty | Password for this Elasticsearch/OpenSearch host. Kept server-side. |
| `hosts[].headers_whitelist` | `[]` | Request headers Cerebro may forward to Elasticsearch, useful behind an authenticating proxy. |
| `auth.session.cookie_max_age_seconds` | `0` | Session cookie max age in seconds. `0` creates a browser-session cookie. |
| `auth.session.max_lifetime_seconds` | `0` | Maximum authenticated session lifetime. `0` disables the limit. |
| `auth.session.idle_timeout_seconds` | `0` | Session idle timeout. `0` disables the limit. |
| `auth.basic.enabled` | `false` | Enables YAML-backed basic authentication. Short form creates provider ID `basic`. |
| `auth.basic.users[].username` | empty | Basic-auth username. |
| `auth.basic.users[].password` | empty | Basic-auth password. Stored only on the backend and checked with bcrypt. |
| `auth.basic.users[].groups` | `[]` | Static groups added to this basic-auth user. |
| `auth.basic.default_groups` | `[]` | Groups added to every user authenticated by this provider. |
| `auth.ldap.enabled` | `false` | Enables LDAP authentication. Short form creates provider ID `ldap`. |
| `auth.ldap.url` | empty | LDAP URL. `ldaps://` is required unless `insecure_ldap` is enabled. |
| `auth.ldap.ca_cert_file` | empty | LDAP CA certificate file for private LDAP trust. |
| `auth.ldap.base_dn` | empty | LDAP base DN for user lookup. |
| `auth.ldap.user_template` | empty | LDAP user DN/search template. |
| `auth.ldap.bind_dn` | empty | Optional LDAP bind DN for searches. |
| `auth.ldap.bind_pw` | empty | Optional LDAP bind password for searches. |
| `auth.ldap.insecure_ldap` | `false` | Allows plain `ldap://` for isolated development tests. Do not use in production. |
| `auth.ldap.default_groups` | `[]` | Groups added to every user authenticated by this provider. |
| `auth.ldap.required_groups` | `[]` | Optional allow-list; login succeeds only when at least one discovered group matches. |
| `auth.ldap.group_search.base_dn` | empty | LDAP group search base DN. |
| `auth.ldap.group_search.user_attr` | empty | LDAP user attribute used for group membership matching. |
| `auth.ldap.group_search.user_attr_template` | empty | LDAP group membership value template. |
| `auth.ldap.group_search.group` | empty | LDAP group filter. |
| `auth.ldap.group_search.name_attr` | `cn` | Attribute exposed as the short group name for RBAC bindings. |
| `auth.proxy.enabled` | `false` | Enables trusted reverse-proxy authentication, for example oauth2-proxy. |
| `auth.proxy.user_header` | empty | Header containing the authenticated username. |
| `auth.proxy.groups_header` | empty | Optional header containing user groups. |
| `auth.proxy.group_separator` | `,` | Separator for `groups_header`. |
| `auth.proxy.default_groups` | `[]` | Groups added to every user authenticated by this provider. |
| `auth.proxy.logout_url` | empty | Optional proxy logout URL used by the frontend logout flow. |
| `auth.proxy.trusted_proxies` | `[]` | Proxy IPs/CIDRs allowed to provide trusted auth headers. |
| `auth.entra_id.enabled` | `false` | Enables Microsoft Entra ID OIDC login. |
| `auth.entra_id.name` | `Microsoft Entra ID` | Login button label. |
| `auth.entra_id.tenant_id` | empty | Entra tenant ID. Either `tenant_id` or `issuer_url` is required. |
| `auth.entra_id.issuer_url` | derived from `tenant_id` | Optional explicit OIDC issuer URL. |
| `auth.entra_id.client_id` | empty | Entra application client ID. |
| `auth.entra_id.client_secret` | empty | Entra application client secret. Kept server-side. |
| `auth.entra_id.redirect_url` | derived from request | Optional exact callback URL registered in Entra ID. |
| `auth.entra_id.username_claim` | `preferred_username` | Claim used as the Cerebro username. |
| `auth.entra_id.groups_claim` | `groups` | Claim used for RBAC groups. |
| `auth.entra_id.default_groups` | `[]` | Groups added to every user authenticated by this provider. |
| `auth.entra_id.scopes` | `["openid", "profile", "email"]` | OIDC scopes requested during login. |
| `auth.oauth.enabled` | `false` | Enables generic OAuth2/OIDC login. |
| `auth.oauth.name` | `OAuth` | Login button label. |
| `auth.oauth.issuer_url` | empty | OIDC discovery issuer URL. Prefer this when supported. |
| `auth.oauth.auth_url` | empty | OAuth2 authorization endpoint for non-discovery providers. |
| `auth.oauth.token_url` | empty | OAuth2 token endpoint for non-discovery providers. |
| `auth.oauth.userinfo_url` | empty | OAuth2 userinfo endpoint for non-discovery providers. |
| `auth.oauth.client_id` | empty | OAuth/OIDC client ID. |
| `auth.oauth.client_secret` | empty | OAuth/OIDC client secret. Kept server-side. |
| `auth.oauth.redirect_url` | derived from request | Optional exact callback URL registered in the provider. |
| `auth.oauth.username_claim` | `preferred_username` | Claim used as the Cerebro username. |
| `auth.oauth.groups_claim` | `groups` | Claim used for RBAC groups. |
| `auth.oauth.default_groups` | `[]` | Groups added to every user authenticated by this provider. |
| `auth.oauth.scopes` | `["openid", "profile", "email"]` | OAuth/OIDC scopes requested during login. |
| `rbac.enabled` | `false` | Enables backend authorization policies. Requires authentication. |
| `rbac.bindings` | `[]` | Subject-to-role bindings, for example user or group to `role:admin`. |
| `rbac.policies` | `[]` | Casbin-backed Cerebro policy rules: subject, resource, action, object and effect. |
| `server.port` | `9000` | HTTP/HTTPS listen port. Can be overridden with `CEREBRO_PORT`. |
| `server.base_path` | `/` | URL path prefix when Cerebro is mounted below `/`. |
| `server.public_url` | empty | External origin used for redirects when Cerebro is behind a reverse proxy. |
| `server.trusted_proxies` | `[]` | Proxy IPs/CIDRs allowed to provide `X-Forwarded-Host` and `X-Forwarded-Proto`. |
| `server.secret` | `change-me` | Session signing secret. Required and must be changed when auth is enabled. Can be overridden with `APPLICATION_SECRET`. |
| `server.cookie_secure` | `true` | Marks session cookies as secure. Keep `true` behind HTTPS; set `false` only for local HTTP development. |
| `server.csrf_enabled` | `true` | Enables session-bound CSRF protection for Cerebro API requests. |
| `server.max_request_bytes` | `5242880` | Maximum accepted Cerebro API request body size. |
| `server.tls_cert_file` | empty | TLS certificate file. Must be set together with `server.tls_key_file`. |
| `server.tls_key_file` | empty | TLS private key file. Must be set together with `server.tls_cert_file`. |
| `server.hsts_enabled` | `true` | Emits HSTS for HTTPS requests, including proxied HTTPS via `X-Forwarded-Proto: https`. |
| `server.hsts_max_age_seconds` | `31536000` | HSTS max-age. |
| `server.hsts_include_subdomains` | `true` | Adds `includeSubDomains` to HSTS. Enable only for domains that should always use HTTPS. |
| `es.gzip` | `true` | Enables gzip for Elasticsearch responses. |
| `es.allow_ad_hoc_hosts` | `false` | Allows users to connect to arbitrary Elasticsearch URLs from the connect page. Keep `false` in shared environments. |
| `es.max_response_bytes` | `26214400` | Maximum Elasticsearch response body size Cerebro will read. |
| `es.ca_cert_file` | empty | Custom CA bundle for Elasticsearch TLS trust. |
| `es.client_cert_file` | empty | Elasticsearch client certificate for mutual TLS. Must be set together with `es.client_key_file`. |
| `es.client_key_file` | empty | Elasticsearch client private key for mutual TLS. Must be set together with `es.client_cert_file`. |
| `es.aws.enabled` | `false` | Enables AWS SigV4 signing for Amazon OpenSearch Service or OpenSearch Serverless. |
| `es.aws.region` | empty | AWS region. Required when `es.aws.enabled` is `true`. |
| `es.aws.service` | `es` when AWS is enabled | SigV4 service name. Use `es` for Amazon OpenSearch Service and `aoss` for OpenSearch Serverless. |
| `es.aws.profile` | empty | AWS shared config profile. |
| `es.aws.access_key_id` | empty | Static AWS access key ID. If omitted, the default AWS credential chain is used. |
| `es.aws.secret_access_key` | empty | Static AWS secret access key. Must be set together with `es.aws.access_key_id`. |
| `es.aws.session_token` | empty | Optional AWS session token for temporary credentials. |
| `rest.history_size` | `50` | Number of REST console requests kept in local history. |
| `features.data_explorer` | `false` | Enables the document browser/editor. Keep disabled unless users are allowed to read and modify index documents. |
| `data.path` | `./cerebro.db` | SQLite file used for REST request history. |
| `logging.level` | `info` | Log level: `debug`, `info`, `warn` or `error`. |
| `logging.format` | `text` | Log format: `text` or `json`. |
| `logging.request_log_enabled` | `true` | Enables per-request HTTP access logs. Request logs are emitted at `info`, so `logging.level: warn` suppresses normal access logs. |
| `logging.auth_log_enabled` | `true` | Enables authentication audit logs. |

Production baseline:

```yaml
hosts:
  - id: "prod"
    name: "Production"
    host: "https://elasticsearch.example.org:9200"
    auth:
      username: "${ES_USERNAME}"
      password: "${ES_PASSWORD}"

auth:
  basic:
    enabled: true
    users:
      - username: "${CEREBRO_USER}"
        password: "${CEREBRO_PASSWORD}"
        groups: ["cerebro-admins"]
      - username: "${CEREBRO_VIEWER_USER}"
        password: "${CEREBRO_VIEWER_PASSWORD}"
        groups: ["cerebro-viewers"]

rbac:
  enabled: true
  bindings:
    - {subject: "group:cerebro-admins", role: "role:admin"}
    - {subject: "group:cerebro-viewers", role: "role:viewer"}
  policies:
    - {subject: "role:admin", resource: "*", action: "*", object: "*", effect: "allow"}
    - {subject: "role:viewer", resource: "*", action: "read", object: "*", effect: "allow"}
    - {subject: "role:viewer", resource: "rest", action: "execute", object: "*", effect: "deny"}

server:
  port: 9000
  base_path: "/"
  secret: "${APPLICATION_SECRET}"
  cookie_secure: true
  csrf_enabled: true
  max_request_bytes: 5242880
  tls_cert_file: "/etc/cerebro/tls/tls.crt"
  tls_key_file: "/etc/cerebro/tls/tls.key"
  hsts_enabled: true
  hsts_max_age_seconds: 31536000
  hsts_include_subdomains: true

es:
  gzip: true
  allow_ad_hoc_hosts: false
  max_response_bytes: 26214400
```

Elasticsearch HTTPS with a custom CA and client certificate:

```yaml
hosts:
  - id: "secure-cluster"
    name: "Secure cluster"
    host: "https://elasticsearch.example.org:9200"
    auth:
      username: "${ES_USERNAME}"
      password: "${ES_PASSWORD}"

es:
  ca_cert_file: "/etc/cerebro/certs/es-ca.pem"
  client_cert_file: "/etc/cerebro/certs/cerebro-client.pem"
  client_key_file: "/etc/cerebro/certs/cerebro-client-key.pem"
  allow_ad_hoc_hosts: false
```

The Elasticsearch TLS settings are global for the Cerebro process and apply to all configured Elasticsearch hosts.

Amazon OpenSearch Service with AWS SigV4 signing:

```yaml
hosts:
  - id: "aws-opensearch"
    name: "AWS OpenSearch"
    host: "https://search-domain.eu-central-1.es.amazonaws.com"

es:
  aws:
    enabled: true
    region: "eu-central-1"
    service: "es"
    # profile: "default"
    # access_key_id: "${AWS_ACCESS_KEY_ID}"
    # secret_access_key: "${AWS_SECRET_ACCESS_KEY}"
    # session_token: "${AWS_SESSION_TOKEN}"
```

If `access_key_id` and `secret_access_key` are omitted, Cerebro uses the default AWS credential chain. Use `service: "aoss"` for OpenSearch Serverless.

Environment variables are expanded inside YAML values. These direct overrides are also supported:

- `CEREBRO_PORT`
- `APPLICATION_SECRET`

## Feature Flags

| Flag | Default | Description |
| --- | --- | --- |
| `features.data_explorer` | `false` | Enables the index document browser/editor. Keep disabled unless Cerebro users are allowed to read and modify index documents. |

## Compatibility

Go Cerebro targets Elasticsearch and OpenSearch clusters through the official Elasticsearch Go client v9 transport path. Docker-backed e2e compatibility tests cover Elasticsearch major versions from 5 to 9 for the core APIs used by Cerebro.

## Authentication

Each auth type supports one short-form provider:

```yaml
auth:
  basic:
    enabled: true
```

For multiple providers of the same type, use named providers:

```yaml
auth:
  basic:
    local_admins:
      enabled: true
    local_viewers:
      enabled: true
  oauth:
    github:
      enabled: true
    dex:
      enabled: true
```

Short-form providers use the auth type as their provider ID (`basic`, `proxy`,
`oauth`, etc.). Named provider IDs must be unique across all auth types and may
contain lowercase letters, digits, `_` and `-`. Do not mix short-form fields and
named providers under the same auth type.

- [Basic auth](./docs/auth-basic.md)
- [Microsoft Entra ID](./docs/auth-entra-id.md)
- [Generic OAuth / OIDC](./docs/auth-oauth.md)
- [LDAP](./docs/auth-ldap.md)
- [Trusted proxy / oauth2-proxy](./docs/auth-proxy.md)
- [RBAC](./docs/RBAC.md)

## Docker

Runtime image:

```sh
docker run --rm -p 9000:9000 \
  -v ./conf/application.yaml:/etc/cerebro/application.yaml:ro \
  ghcr.io/piotrkochan/go-cerebro:latest \
  serve -config /etc/cerebro/application.yaml
```

The development stack uses:

- `elasticsearch`
- `elasticsearch-2`
- `cerebro`
- `frontend`

Start it with:

```sh
docker compose up --build
```

Persistent Elasticsearch data is stored in Docker volumes. To reset only the containers without deleting indices:

```sh
docker compose down
docker compose up --build
```

More configuration examples are in [examples](./examples), including basic auth, LDAP, OAuth/OIDC and Elasticsearch mutual TLS.

## Security Notes

Cerebro can manage Elasticsearch clusters. Treat access to this UI as administrative access.

- Serve Cerebro over HTTPS, either with `server.tls_cert_file`/`server.tls_key_file` or through a reverse proxy.
- Enable at least one auth provider outside local development.
- Set `server.secret`.
- Keep `es.allow_ad_hoc_hosts: false` unless you explicitly need user-supplied ES targets.
- Use dedicated Elasticsearch users with the minimum required privileges.
- Use `ldaps://` or `auth.ldap.ca_cert_file` for LDAP trust.
- Use `auth.proxy` only when Cerebro is reachable exclusively through the configured trusted proxies.
- Do not put Elasticsearch credentials into host URLs; use the `auth` block per host.

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md).

## License

MIT, same as the original Cerebro project. See [LICENSE](./LICENSE).
