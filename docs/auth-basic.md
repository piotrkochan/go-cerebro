# Basic Auth

Basic auth is configured in YAML and stores users only on the Cerebro backend.
The frontend never receives Elasticsearch credentials.

```yaml
auth:
  basic:
    enabled: true
    default_groups: ["cerebro-viewers"]
    users:
      - username: "${CEREBRO_ADMIN_USER}"
        password: "${CEREBRO_ADMIN_PASSWORD}"
        groups: ["cerebro-admins"]
      - username: "${CEREBRO_VIEWER_USER}"
        password: "${CEREBRO_VIEWER_PASSWORD}"
        groups: ["cerebro-viewers"]

server:
  secret: "${APPLICATION_SECRET}"
```

`default_groups` are added to every user from this provider. Per-user `groups` are added too.
Groups are optional. Use them with RBAC bindings:

```yaml
rbac:
  enabled: true
  bindings:
    - {subject: "group:cerebro-admins", role: "role:admin"}
    - {subject: "group:cerebro-viewers", role: "role:viewer"}
  policies:
    - {subject: "role:admin", resource: "*", action: "*", object: "*", effect: "allow"}
    - {subject: "role:viewer", resource: "*", action: "read", object: "*", effect: "allow"}
```
