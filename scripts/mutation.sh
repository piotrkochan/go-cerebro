#!/usr/bin/env bash
set -euo pipefail

go run github.com/go-gremlins/gremlins/cmd/gremlins@v0.6.0 unleash . \
  --integration \
  --threshold-efficacy "${GREMLINS_THRESHOLD_EFFICACY:-100}" \
  --workers "${GREMLINS_WORKERS:-2}"
