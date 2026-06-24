# OpenLDAP Group Auth Example

This example runs Go Cerebro with LDAP authentication and an additional group membership check.

Start it from this directory:

```sh
docker compose up --build
```

Load the test user and group:

```sh
./add_group.sh
```

Open `http://localhost:9000` and log in with:

- username: `test`
- password: `test`

The active Cerebro configuration is [application.yaml](./application.yaml). The important part is `auth.settings.group_search`, which requires the user to match:

```text
memberof=cn=cerebro,ou=groups,dc=example,dc=org
```

This example intentionally sets `insecure_ldap: true` because the test container uses plain `ldap://`. Production LDAP should use `ldaps://` or `auth.settings.ca_cert_file`.
