# Elasticsearch Mutual TLS Example

This example shows how to configure Go Cerebro for an Elasticsearch cluster exposed over HTTPS with a custom CA and a client certificate.

Copy [application.yaml](./application.yaml), update the host, credentials and certificate paths, then run Cerebro with:

```sh
cerebro serve -config application.yaml
```

The certificate files must be readable by the Cerebro process:

- `/etc/cerebro/certs/es-ca.pem`: CA certificate used to verify Elasticsearch.
- `/etc/cerebro/certs/cerebro-client.pem`: client certificate presented to Elasticsearch.
- `/etc/cerebro/certs/cerebro-client-key.pem`: private key for the client certificate.

The `es.*_cert_file` settings are global for the Cerebro process and apply to all configured Elasticsearch hosts.
