# Go Cerebro

[![CI](https://github.com/piotrkochan/go-cerebro/actions/workflows/ci.yml/badge.svg)](https://github.com/piotrkochan/go-cerebro/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/piotrkochan/go-cerebro)](./LICENSE)

Go Cerebro is a fork of the original [lmenezes/cerebro](https://github.com/lmenezes/cerebro) Elasticsearch web admin tool. The original application was Java + Angular; this fork is rewritten to Go, Huma, React and TypeScript while keeping the Cerebro user experience. AI assistance was used during the refactor; despite careful review, there may still be rough edges, so please keep that in mind while evaluating or using this version.

The backend exposes a Huma/OpenAPI HTTP API. The frontend consumes a generated TypeScript client from `openapi/cerebro.json`; hand-written frontend API wrappers should be avoided.

## Status

This fork is under active rewrite. Prefer current code and this README over old upstream instructions, especially anything mentioning Play Framework, Angular, `-D...` JVM flags or committed frontend build bundles.

## Requirements

- Go 1.26.x
- Node.js 24+ or 26+
- npm
- Docker Compose, for the local Elasticsearch development stack

## Quick Start

Run the full development stack:

```sh
docker compose up --build
```

Then open:

- Application served by Go: `http://localhost:9000`
- Vite dev frontend with live reload: `http://localhost:5173`
- Local Elasticsearch: `http://localhost:9200`

The development config is [conf/application.dev.yaml](./conf/application.dev.yaml). It connects to the Elasticsearch services defined in [docker-compose.yaml](./docker-compose.yaml) and enables development-only features such as the data explorer.

## Local Development

Install frontend dependencies:

```sh
npm ci
```

Generate the OpenAPI document and TypeScript client:

```sh
npm run api:generate
```

Run the Go server:

```sh
go run ./cmd/cerebro serve -config conf/application.dev.yaml
```

Run the Vite frontend in another terminal:

```sh
npm run dev
```

Useful checks:

```sh
npm run typecheck
npm run api:typecheck
go test ./...
npm test
```

Build everything:

```sh
npm run build
go build -o cerebro ./cmd/cerebro
```

## CLI

Serve the application:

```sh
cerebro serve -config conf/application.yaml -public public
```

Generate the OpenAPI spec:

```sh
cerebro openapi -config conf/application.example.yaml > openapi/cerebro.json
```

Print the version:

```sh
cerebro version
```

Release builds inject the version with:

```sh
go build -ldflags="-X github.com/lmenezes/cerebro/internal/version.Version=0.10.0" ./cmd/cerebro
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

Environment variables are expanded inside YAML values. These direct overrides are also supported:

- `CEREBRO_PORT`
- `APPLICATION_SECRET`
- `AUTH_TYPE`

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

## API And Frontend Client

The API is registered in Go through Huma. OpenAPI is generated from the backend routes:

```sh
npm run api:openapi
```

The TypeScript client is generated with `@hey-api/openapi-ts`:

```sh
npm run api:generate
```

Generated files live in:

- [openapi/cerebro.json](./openapi/cerebro.json)
- [src/api/client](./src/api/client)

When a backend API shape changes, regenerate the client and commit the changed OpenAPI/client files. Do not add custom frontend response adapters unless the API contract itself is wrong.

## Docker

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
