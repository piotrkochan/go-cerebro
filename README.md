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
cerebro openapi -config conf/application.example.yaml > openapi/cerebro.json
```

Print the version:

```sh
cerebro version
```

## Configuration

Copy [conf/application.example.yaml](./conf/application.example.yaml) to `conf/application.yaml` and edit it for your environment.

Important sections:

- `hosts`: known Elasticsearch clusters. Optional `hosts[].id` becomes the stable cluster slug used in URLs and RBAC; use lowercase letters, digits and hyphens. If omitted, the slug is generated from `hosts[].name`. Keep `es.allow_ad_hoc_hosts: false` in shared environments.
- `hosts[].headers_whitelist`: request headers that Cerebro may forward to Elasticsearch, useful behind an authenticating proxy.
- `auth.basic`, `auth.ldap`, `auth.proxy`, `auth.entra_id`: optional authentication providers. Enable at least one outside local development.
- `auth.session`: optional cookie lifetime, max session lifetime and idle timeout settings.
- `rbac`: optional YAML authorization policies for users and groups.
- `server.base_path`: URL path prefix when Cerebro is mounted below `/`.
- `server.public_url`: external origin used for redirects when Cerebro is behind a reverse proxy.
- `server.trusted_proxies`: proxy IPs/CIDRs allowed to provide `X-Forwarded-Host` and `X-Forwarded-Proto`.
- `server.secret`: required for authenticated deployments. Set it to a strong random value.
- `server.cookie_secure`: keep `true` behind HTTPS.
- `server.csrf_enabled`: session-bound CSRF protection for browser requests. Keep enabled for browser-facing deployments; it is not a replacement for authentication.
- `server.max_request_bytes`: maximum accepted Cerebro API request body size.
- `server.tls_cert_file`, `server.tls_key_file`: optional HTTPS listener certificate and private key.
- `server.hsts_enabled`, `server.hsts_max_age_seconds`, `server.hsts_include_subdomains`: HTTPS Strict Transport Security settings. Enable only for domains that should always use HTTPS.
- `es.gzip`: enable gzip for Elasticsearch responses.
- `es.max_response_bytes`: maximum Elasticsearch response body size Cerebro will read.
- `es.aws`: AWS SigV4 signing for Amazon OpenSearch Service and OpenSearch Serverless.
- `es.ca_cert_file`, `es.client_cert_file`, `es.client_key_file`: TLS trust and mutual TLS for Elasticsearch.
- `auth.ldap.ca_cert_file`: custom LDAP CA trust.
- `rest.history_size`: number of REST console requests kept in local history.
- `features.data_explorer`: document browser/editor. Disabled by default because it exposes index data to authenticated users.
- `data.path`: SQLite file used for REST request history.
- `logging.level`, `logging.format`, `logging.request_log_enabled`, `logging.auth_log_enabled`: application log level/format, per-request HTTP access logs and auth audit logs. Request logs are emitted at `info`, so `logging.level: warn` also suppresses normal access logs.

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

- [Basic auth](./docs/auth-basic.md)
- [Microsoft Entra ID](./docs/auth-entra-id.md)
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

More configuration examples are in [examples](./examples), including basic auth, LDAP and Elasticsearch mutual TLS.

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
