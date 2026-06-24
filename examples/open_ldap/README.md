# OpenLDAP Auth Example

This example runs Go Cerebro with LDAP authentication and a local OpenLDAP container.

Start it from this directory:

```sh
docker compose up --build
```

Add the test user:

```sh
docker exec ldap ldapadd -x -D "cn=admin,dc=example,dc=org" -w admin -f /opt/test-user.ldif -H ldap://localhost -ZZ
```

Open `http://localhost:9000` and log in with:

- username: `test`
- password: `test`

The active Cerebro configuration is [application.yaml](./application.yaml). This example intentionally sets `insecure_ldap: true` because the test container uses plain `ldap://`. Production LDAP should use `ldaps://` or `auth.settings.ca_cert_file`.
