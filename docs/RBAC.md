# RBAC

Go Cerebro can enforce backend authorization with YAML policies. Authentication identifies the user and groups. RBAC decides which API operations are allowed.

The policy engine is Casbin. The YAML shape is kept Cerebro-specific: policies target Cerebro features and actions instead of raw Elasticsearch endpoints.

RBAC is disabled by default.

```yaml
rbac:
  enabled: true
  default_role: "role:viewer"
  bindings:
    - {subject: "admin", role: "role:admin"}
    - {subject: "group:cerebro-admins", role: "role:admin"}
  policies:
    - {subject: "role:admin", resource: "*", action: "*", object: "*", effect: "allow"}
    - {subject: "role:viewer", resource: "*", action: "read", object: "*", effect: "allow"}
    - {subject: "role:viewer", resource: "rest", action: "execute", object: "*", effect: "deny"}
```

## Model

Policy fields:

- `subject`: user, group or role name. Roles conventionally use `role:name`.
- `resource`: API resource, for example `overview`, `indices`, `templates`, `rest`, `cluster_settings`, `data_streams`, `ilm`.
- `action`: operation type. Common values are `read`, `create`, `write`, `delete`, `execute`.
- `object`: cluster-scoped target. Cluster-wide operations use the cluster id, resource operations use `cluster-id/name`, for example `prod` or `prod/index-*`.
- `effect`: `allow` or `deny`.

Wildcards use shell-style matching. `deny` wins over `allow`, including denies on one principal overriding allows inherited from another principal or default role.

## Resources

Only resources listed in this table are part of the supported RBAC model.

| Resource | Actions | Object format | Covers |
| --- | --- | --- | --- |
| `overview` | `read` | `cluster` | Overview data. |
| `nodes` | `read` | `cluster` or `cluster/node` | Nodes page and node details used by pages. |
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

## Subjects

Users are matched by username:

```yaml
bindings:
  - {subject: "alice", role: "role:admin"}
```

Groups are matched with the `group:` prefix:

```yaml
bindings:
  - {subject: "group:cerebro-admins", role: "role:admin"}
```

LDAP group discovery exposes both the short group name from `auth.ldap.group_search.name_attr` and the full group DN. Either can be used:

```yaml
bindings:
  - {subject: "group:cerebro-admins", role: "role:admin"}
  - {subject: "group:cn=cerebro-admins,ou=groups,dc=example,dc=org", role: "role:admin"}
```

Proxy auth groups come from `auth.proxy.groups_header`, split by `auth.proxy.group_separator`.

Basic auth can attach static groups:

```yaml
auth:
  basic:
    enabled: true
    username: "${BASIC_AUTH_USER}"
    password: "${BASIC_AUTH_PWD}"
    groups: ["cerebro-admins"]
```

## Per-Cluster Permissions

Configured clusters can define a stable id:

```yaml
hosts:
  - id: "prod"
    name: "Production EU"
    host: "https://elasticsearch.example.org:9200"
```

The configured `id` becomes the cluster slug used in URLs and RBAC objects. It must contain only lowercase ASCII letters, digits and hyphens, for example `prod` or `production-eu`.

If `id` is omitted, Cerebro generates the slug from `name`; for example `Production EU` becomes `production-eu`.

Every cluster-scoped request sets `object` to the cluster id or to a value prefixed with the cluster id. This means permissions can be granted globally, per cluster, or per resource inside a cluster:

- `object: "*"`: all clusters.
- `object: "prod"`: cluster-wide operations on the `prod` cluster.
- `object: "prod/*"`: all resource-level operations inside the `prod` cluster.
- `object: "prod/index-*"`: only matching index-like objects inside the `prod` cluster.

Allow read-only access to one cluster:

```yaml
rbac:
  enabled: true
  bindings:
    - {subject: "group:ops-readonly", role: "role:prod-reader"}
  policies:
    - {subject: "role:prod-reader", resource: "*", action: "read", object: "production-eu", effect: "allow"}
    - {subject: "role:prod-reader", resource: "*", action: "read", object: "production-eu/*", effect: "allow"}
```

Allow data explorer edits only on development indices:

```yaml
rbac:
  enabled: true
  bindings:
    - {subject: "group:devs", role: "role:dev-editor"}
  policies:
    - {subject: "role:dev-editor", resource: "*", action: "read", object: "dev*", effect: "allow"}
    - {subject: "role:dev-editor", resource: "documents", action: "*", object: "dev/index-*", effect: "allow"}
```

## Recommended Roles

A practical starting point:

- `role:admin`: full access.
- `role:operator`: read access plus operational actions such as snapshots, repositories, allocation and settings.
- `role:viewer`: read-only access.

Keep `rest execute` restricted. The REST console can call Elasticsearch APIs that are broader than the visible Cerebro screens.
