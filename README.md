# Go Cerebro

[![CI](https://github.com/piotrkochan/go-cerebro/actions/workflows/ci.yml/badge.svg)](https://github.com/piotrkochan/go-cerebro/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/piotrkochan/go-cerebro)](./LICENSE)

Go Cerebro is a fork of the original [lmenezes/cerebro](https://github.com/lmenezes/cerebro) Elasticsearch web admin tool. The original application was Java + Angular; this fork is rewritten to Go, Huma, React and TypeScript while keeping the Cerebro user experience. AI assistance was used during the refactor; despite careful review, there may still be rough edges, so please keep that in mind while evaluating or using this version.

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
cerebro openapi -config conf/application.example.yaml > openapi/cerebro.json
```

Print the version:

```sh
cerebro version
```

## Configuration

Copy [conf/application.example.yaml](./conf/application.example.yaml) to `conf/application.yaml` and edit it for your environment.

Important sections:

- `hosts`: known Elasticsearch clusters. Keep `es.allow_ad_hoc_hosts: false` in shared environments.
- `auth`: `disabled`, `basic` or `ldap`. Do not expose an instance with `auth.type: disabled`.
- `server.secret`: required for authenticated deployments. Set it to a strong random value.
- `server.cookie_secure`: keep `true` behind HTTPS.
- `server.hsts_enabled`, `server.hsts_max_age_seconds`, `server.hsts_include_subdomains`: HTTPS Strict Transport Security settings. Enable only for domains that should always use HTTPS.
- `es.ca_cert_file`, `es.client_cert_file`, `es.client_key_file`: TLS trust and mutual TLS for Elasticsearch.
- `auth.settings.ca_cert_file`: custom LDAP CA trust.
- `features.data_explorer`: document browser/editor. Disabled by default because it exposes index data to authenticated users.
- `data.path`: SQLite file used for REST request history.

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
  secret: "${APPLICATION_SECRET}"
  cookie_secure: true
  hsts_enabled: true
  hsts_max_age_seconds: 31536000
  hsts_include_subdomains: true

es:
  allow_ad_hoc_hosts: false
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

- Prefer a reverse proxy with HTTPS.
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
