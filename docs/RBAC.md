# RBAC

Go Cerebro can enforce backend authorization with RBAC policies written in `application.yaml`.

Authentication answers: "who is this user and which groups do they have?" RBAC answers:
"which Cerebro features may this user use?"

RBAC is disabled by default. When it is enabled, every protected API request must match at least one
`allow` policy. A matching `deny` policy always wins.

Go Cerebro uses Casbin as the policy engine, but the configuration is intentionally Cerebro-specific:
policies target Cerebro features such as `indices`, `templates`, `data_streams` and `rest`, not raw
Elasticsearch endpoints.

## Quick Start

Give every configured cluster a stable `id`, bind users or groups to roles, then grant permissions to
those roles:

```yaml
hosts:
  - id: "prod"
    name: "Production EU"
    host: "https://elasticsearch.example.org:9200"

rbac:
  enabled: true
  default_role: "role:viewer"

  bindings:
    - {subject: "admin", role: "role:admin"}
    - {subject: "group:cerebro-admins", role: "role:admin"}

  policies:
    - {subject: "role:admin", resource: "*", action: "*", object: "*", effect: "allow"}
    - {subject: "role:viewer", resource: "*", action: "read", object: "prod", effect: "allow"}
    - {subject: "role:viewer", resource: "*", action: "read", object: "prod/*", effect: "allow"}
    - {subject: "role:viewer", resource: "rest", action: "execute", object: "*", effect: "deny"}
```

This example gives every authenticated user read-only access to the `prod` cluster through
`default_role`, gives admins full access, and blocks REST-console execution for viewers.

## Policy Model

Each policy has five fields:

| Field | Meaning |
| --- | --- |
| `subject` | User, group or role receiving the rule. Roles conventionally use `role:name`. |
| `resource` | Cerebro feature being accessed, for example `indices`, `templates`, `rest` or `ilm`. |
| `action` | Operation on that feature, for example `read`, `write`, `delete` or `execute`. |
| `object` | Cluster-scoped target such as `prod`, `prod/*` or `prod/index-*`. |
| `effect` | `allow` or `deny`. |

Policies are evaluated like this:

- RBAC disabled: request is allowed.
- RBAC enabled: at least one matching `allow` is required.
- `deny` wins over `allow`, even if the allow comes from another role or from `default_role`.
- Unknown resources and invalid actions are rejected when the config is loaded.

## Subjects And Roles

The authenticated username is available as both `alice` and `user:alice`.

Groups are available as both the raw group name and `group:<name>`. Prefer the `group:` form in
configuration because it is explicit and avoids collisions with usernames.

`default_role` is optional. When it is set, it is applied to every authenticated user. When it is
omitted, users only get permissions from direct policies or roles attached through `bindings`.

Use `bindings` to attach users or groups to roles:

```yaml
rbac:
  bindings:
    - {subject: "alice", role: "role:admin"}
    - {subject: "user:bob", role: "role:operator"}
    - {subject: "group:cerebro-viewers", role: "role:viewer"}
```

Policies normally target roles:

```yaml
rbac:
  policies:
    - {subject: "role:viewer", resource: "*", action: "read", object: "*", effect: "allow"}
```

Direct user or group policies also work, but roles keep the config easier to maintain.

## Groups From Auth Providers

Basic auth can attach static groups:

```yaml
auth:
  basic:
    enabled: true
    users:
      - username: "${BASIC_AUTH_USER}"
        password: "${BASIC_AUTH_PWD}"
        groups: ["cerebro-admins"]
```

LDAP group discovery exposes the short group name from `auth.ldap.group_search.name_attr` and the
full group DN:

```yaml
rbac:
  bindings:
    - {subject: "group:cerebro-admins", role: "role:admin"}
    - {subject: "group:cn=cerebro-admins,ou=groups,dc=example,dc=org", role: "role:admin"}
```

Proxy auth groups come from `auth.proxy.groups_header`, split by `auth.proxy.group_separator`.

Microsoft Entra ID groups come from the configured `auth.entra_id.groups_claim`. Some tenants emit
group object IDs instead of names; in that case bind the object ID as `group:<id>` or use app roles.

## Cluster IDs And Objects

Every configured cluster should have a stable `id`:

```yaml
hosts:
  - id: "prod"
    name: "Production EU"
    host: "https://elasticsearch.example.org:9200"
```

The `id` is used in URLs and in RBAC `object` values. It must contain only lowercase ASCII letters,
digits and single hyphens, for example `prod` or `production-eu`.

If `id` is omitted, Cerebro generates the cluster slug from `name`. For example, `Production EU`
becomes `production-eu`. Explicit `id` is recommended for shared environments because generated slugs
can change when the display name changes.

Object patterns:

| Pattern | Matches |
| --- | --- |
| `*` | All clusters and resource-level objects. |
| `prod` | Cluster-wide requests for the `prod` cluster. |
| `prod/*` | Resource-level requests inside the `prod` cluster. |
| `prod/*/*` | Two-level resource objects, for example `prod/repository/snapshot`. |
| `prod/index-*` | Index-like objects in `prod` whose name starts with `index-`. |

Use both `prod` and `prod/*` when a role should read a whole cluster and most resource pages:

```yaml
policies:
  - {subject: "role:prod-reader", resource: "*", action: "read", object: "prod", effect: "allow"}
  - {subject: "role:prod-reader", resource: "*", action: "read", object: "prod/*", effect: "allow"}
```

## Wildcards

Wildcards use shell-style matching from Go's `path.Match`.

Important consequences:

- `*` does not cross `/`.
- `prod/*` matches `prod/index-a`.
- `prod/*` does not match `prod/repository/snapshot` because that object has two `/` separators.
- Use `prod/*/*` when a feature uses two-level objects, for example snapshot operations on
  `repository/snapshot`.
- `prod*` matches cluster IDs such as `prod` or `prod-eu`, but not `prod/index-a`.

Prefer explicit patterns such as `prod`, `prod/*` and `prod/index-*`.

## Resources

Only resources listed below are supported.

| Resource | Actions | Object format | Covers |
| --- | --- | --- | --- |
| `overview` | `read` | `cluster` | Overview data. |
| `nodes` | `read` | `cluster` or `cluster/node` | Nodes page and node details. |
| `indices` | `read`, `create`, `write`, `delete`, `refresh`, `close`, `open`, `clear_cache`, `force_merge` | `cluster` or `cluster/index` | Index create/delete/open/close/refresh/cache/force-merge/settings helpers. |
| `documents` | `read`, `write` | `cluster/index` | Data explorer search, insert and edit. |
| `rest` | `read`, `execute` | `cluster` | REST console history and Elasticsearch request execution. |
| `analysis` | `read`, `execute` | `cluster` or `cluster/index` | Analysis page metadata and analyze calls. |
| `cat` | `read` | `cluster` or `cluster/api` | Cat APIs. |
| `aliases` | `read`, `write`, `delete` | `cluster` | Alias management. |
| `templates` | `read`, `write`, `delete` | `cluster` or `cluster/template` | Index, component and legacy templates. |
| `repositories` | `read`, `write`, `delete` | `cluster` or `cluster/repository` | Snapshot repository management. |
| `snapshots` | `read`, `write`, `delete`, `restore` | `cluster`, `cluster/repository` or `cluster/repository/snapshot` | Snapshot list/create/delete/restore. |
| `data_streams` | `read`, `write`, `delete`, `rollover`, `update_lifecycle`, `attach_ilm`, `detach_ilm` | `cluster` or `cluster/data-stream` | Data stream management. |
| `ilm` | `read`, `write`, `delete` | `cluster` or `cluster/policy` | ILM policy management. |
| `cluster_settings` | `read`, `write` | `cluster` | Cluster settings page. |
| `shard_allocation` | `enable`, `disable` | `cluster` | Overview shard allocation toggle. |
| `shards` | `relocate` | `cluster/index` | Shard relocation. |

## Example Roles

### Read-Only Viewer

```yaml
rbac:
  enabled: true
  default_role: "role:viewer"
  policies:
    - {subject: "role:viewer", resource: "*", action: "read", object: "prod", effect: "allow"}
    - {subject: "role:viewer", resource: "*", action: "read", object: "prod/*", effect: "allow"}
    - {subject: "role:viewer", resource: "rest", action: "execute", object: "*", effect: "deny"}
```

### Data Explorer Editor For One Index Prefix

```yaml
rbac:
  enabled: true
  bindings:
    - {subject: "group:devs", role: "role:dev-editor"}
  policies:
    - {subject: "role:dev-editor", resource: "*", action: "read", object: "dev", effect: "allow"}
    - {subject: "role:dev-editor", resource: "*", action: "read", object: "dev/*", effect: "allow"}
    - {subject: "role:dev-editor", resource: "documents", action: "write", object: "dev/app-*", effect: "allow"}
```

### Operator Without REST Console Execution

```yaml
rbac:
  enabled: true
  bindings:
    - {subject: "group:operators", role: "role:operator"}
  policies:
    - {subject: "role:operator", resource: "*", action: "read", object: "prod", effect: "allow"}
    - {subject: "role:operator", resource: "*", action: "read", object: "prod/*", effect: "allow"}
    - {subject: "role:operator", resource: "indices", action: "refresh", object: "prod/*", effect: "allow"}
    - {subject: "role:operator", resource: "indices", action: "clear_cache", object: "prod/*", effect: "allow"}
    - {subject: "role:operator", resource: "snapshots", action: "*", object: "prod/*", effect: "allow"}
    - {subject: "role:operator", resource: "snapshots", action: "*", object: "prod/*/*", effect: "allow"}
    - {subject: "role:operator", resource: "repositories", action: "*", object: "prod/*", effect: "allow"}
    - {subject: "role:operator", resource: "shards", action: "relocate", object: "prod/*", effect: "allow"}
    - {subject: "role:operator", resource: "rest", action: "execute", object: "*", effect: "deny"}
```

## Notes

- RBAC is backend authorization. Hiding or showing frontend buttons is not a security boundary.
- The REST console can call Elasticsearch APIs outside the visible Cerebro screens. Keep
  `rest execute` restricted.
- Cerebro RBAC does not replace Elasticsearch security. Elasticsearch should still enforce its own
  users, roles and TLS where appropriate.
