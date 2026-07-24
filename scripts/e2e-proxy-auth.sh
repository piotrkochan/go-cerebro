#!/usr/bin/env bash
set -euo pipefail

go test -tags=e2e ./internal/e2e -run TestProxyAuthThroughTrustedReverseProxy -count=1 -v
