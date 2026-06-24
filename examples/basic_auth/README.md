# Basic Auth Example

This example runs Go Cerebro with HTTP basic auth enabled.

Start it from this directory:

```sh
docker compose up --build
```

Open `http://localhost:9000` and log in with:

- username: `admin`
- password: `admin`

The active configuration is [application.yaml](./application.yaml). It shows the current YAML format used by this fork; the old upstream `-D...` JVM flags are not used.
