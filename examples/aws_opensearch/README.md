# AWS OpenSearch SigV4 Example

This example shows how to configure Go Cerebro for Amazon OpenSearch Service with AWS Signature Version 4 request signing.

Copy [application.yaml](./application.yaml), update the OpenSearch endpoint and region, then run:

```sh
cerebro serve -config application.yaml
```

By default Cerebro uses the standard AWS credential chain:

- environment variables,
- shared AWS config/credentials files,
- container or instance role credentials.

For OpenSearch Serverless, set `es.aws.service` to `aoss`.
