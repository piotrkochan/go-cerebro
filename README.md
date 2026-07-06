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
  - name: "Local cluster"
    host: "http://localhost:9200"

auth:
  type: "disabled"

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
| `AUTH_TYPE` | `auth.type` |

| Option | Default | Description |
| --- | --- | --- |
| `hosts` | `[]` | Known Elasticsearch/OpenSearch clusters shown on the connect page. |
| `hosts[].name` | `hosts[].host` | Display name and source for the stable cluster slug used in API routes. |
| `hosts[].host` | required | Elasticsearch/OpenSearch HTTP URL. Credentials in the URL are rejected; use `hosts[].auth`. |
| `hosts[].auth.username` | empty | Username for this Elasticsearch/OpenSearch host. Kept server-side. |
| `hosts[].auth.password` | empty | Password for this Elasticsearch/OpenSearch host. Kept server-side. |
| `hosts[].headers_whitelist` | `[]` | Request headers Cerebro may forward to Elasticsearch, useful behind an authenticating proxy. |
| `auth.type` | `disabled` | Application login provider: `disabled`, `none`, `basic` or `ldap`. Do not expose shared instances with auth disabled. |
| `auth.settings.username` | empty | Basic-auth username. |
| `auth.settings.password` | empty | Basic-auth password. |
| `auth.settings.url` | empty | LDAP URL. `ldaps://` is required unless `insecure_ldap` is enabled. |
| `auth.settings.ca_cert_file` | empty | LDAP CA certificate file for private LDAP trust. |
| `auth.settings.base_dn` | empty | LDAP base DN for user lookup. |
| `auth.settings.method` | empty | LDAP authentication method used by the LDAP service. |
| `auth.settings.user_template` | empty | LDAP user DN/search template. |
| `auth.settings.bind_dn` | empty | LDAP bind DN for searches. |
| `auth.settings.bind_pw` | empty | LDAP bind password for searches. |
| `auth.settings.insecure_ldap` | `false` | Allows plain `ldap://` for isolated development tests. Do not use in production. |
| `auth.settings.group_search.base_dn` | empty | LDAP group search base DN. |
| `auth.settings.group_search.user_attr` | empty | LDAP user attribute used for group membership matching. |
| `auth.settings.group_search.user_attr_template` | empty | LDAP group membership value template. |
| `auth.settings.group_search.group` | empty | Required LDAP group DN/name. |
| `server.port` | `9000` | HTTP/HTTPS listen port. Can be overridden with `CEREBRO_PORT`. |
| `server.base_path` | `/` | URL path prefix when Cerebro is mounted below `/`. |
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

Production baseline:

```yaml
hosts:
  - name: "Production"
    host: "https://elasticsearch.example.org:9200"
    auth:
      username: "${ES_USERNAME}"
      password: "${ES_PASSWORD}"

auth:
  type: "basic"
  settings:
    username: "${CEREBRO_USER}"
    password: "${CEREBRO_PASSWORD}"

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
  - name: "Secure cluster"
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
  - name: "AWS OpenSearch"
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
- `AUTH_TYPE`

## Feature Flags

| Flag | Default | Description |
| --- | --- | --- |
| `features.data_explorer` | `false` | Enables the index document browser/editor. Keep disabled unless Cerebro users are allowed to read and modify index documents. |

## Compatibility

Go Cerebro targets Elasticsearch and OpenSearch clusters through the official Elasticsearch Go client v9 transport path. Docker-backed e2e compatibility tests cover Elasticsearch major versions from 5 to 9 for the core APIs used by Cerebro.

## Authentication

Basic auth example:

```yaml
auth:
  type: "basic"
  settings:
    username: "${BASIC_AUTH_USER}"
    password: "${BASIC_AUTH_PWD}"
server:
  secret: "${APPLICATION_SECRET}"
```

LDAP uses `ldaps://` by default. For a private test-only LDAP server you can set `insecure_ldap: true`, but do not use that in production.

```yaml
auth:
  type: "ldap"
  settings:
    url: "ldaps://ldap.example.org:636"
    ca_cert_file: "/etc/cerebro/ldap-ca.pem"
    base_dn: "ou=people,dc=example,dc=org"
    method: "simple"
    user_template: "uid=%s,%s"
    bind_dn: "cn=readonly,dc=example,dc=org"
    bind_pw: "${LDAP_BIND_PWD}"
```

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
- Keep `auth.type` enabled outside local development.
- Set `server.secret`.
- Keep `es.allow_ad_hoc_hosts: false` unless you explicitly need user-supplied ES targets.
- Use dedicated Elasticsearch users with the minimum required privileges.
- Use `ldaps://` or `auth.settings.ca_cert_file` for LDAP trust.
- Do not put Elasticsearch credentials into host URLs; use the `auth` block per host.

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md).

## License

MIT, same as the original Cerebro project. See [LICENSE](./LICENSE).
