# LDAP Authentication

Go Cerebro can authenticate users against LDAP. Production deployments should use `ldaps://`.

## Config

```yaml
auth:
  ldap:
    enabled: true
    url: "ldaps://ldap.example.org:636"
    ca_cert_file: "/etc/cerebro/ldap-ca.pem"
    base_dn: "ou=people,dc=example,dc=org"
    method: "simple"
    user_template: "uid=%s,%s"
    bind_dn: "cn=readonly,dc=example,dc=org"
    bind_pw: "${LDAP_BIND_PWD}"
    required_groups: ["cerebro-admins"]
    group_search:
      base_dn: "ou=groups,dc=example,dc=org"
      user_attr: "member"
      user_attr_template: "uid=%s,ou=people,dc=example,dc=org"
      group: "(objectClass=groupOfNames)"
      name_attr: "cn"

server:
  secret: "${APPLICATION_SECRET}"
```

`user_template` receives two values: username and `base_dn`.

`group_search.user_attr_template` receives the same values and should produce the user value stored in group membership entries.

## RBAC Groups

When `group_search` is configured, Cerebro exposes both:

- the full group DN
- the short group name from `group_search.name_attr`

Both can be bound in RBAC:

```yaml
rbac:
  enabled: true
  bindings:
    - {subject: "group:cerebro-admins", role: "role:admin"}
    - {subject: "group:cn=cerebro-admins,ou=groups,dc=example,dc=org", role: "role:admin"}
```

`required_groups` is optional. When set, login succeeds only if LDAP group discovery returns at least one listed group.

## Local Examples

The `examples/open_ldap` and `examples/open_ldap_group` directories use plain `ldap://` with `insecure_ldap: true` for local testing only.

## Security Notes

- Use `ldaps://` in production.
- Use `auth.ldap.ca_cert_file` when the LDAP server uses a private CA.
- Do not use `insecure_ldap: true` outside isolated local tests.
- Use a read-only bind account for group discovery.
- Use `required_groups` when only specific LDAP groups should be able to sign in.
