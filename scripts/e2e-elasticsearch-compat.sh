#!/usr/bin/env bash
set -euo pipefail

versions=(
  "5:5.6.16"
  "6:6.8.23"
  "7:7.10.2"
  "7:7.17.28"
  "8:8.15.0"
  "8:8.19.6"
  "9:9.2.0"
  "9:9.3.6"
)

if [[ "${CEREBRO_E2E_ES_VERSIONS:-}" != "" ]]; then
  IFS=" " read -r -a versions <<< "${CEREBRO_E2E_ES_VERSIONS}"
fi

cleanup() {
  if [[ "${container:-}" != "" ]]; then
    docker rm -f "${container}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

wait_for_elasticsearch() {
  local url="$1"
  local deadline=$((SECONDS + 180))
  until curl -fsS "${url}" >/dev/null 2>&1; do
    if (( SECONDS >= deadline )); then
      docker logs "${container}" >&2 || true
      echo "Elasticsearch did not start at ${url}" >&2
      return 1
    fi
    sleep 2
  done
}

for entry in "${versions[@]}"; do
  major="${entry%%:*}"
  tag="${entry#*:}"
  image="docker.elastic.co/elasticsearch/elasticsearch:${tag}"
  container="go-cerebro-e2e-es-${major}-${RANDOM}"

  echo "==> Elasticsearch ${tag}"
  docker rm -f "${container}" >/dev/null 2>&1 || true
  docker run -d \
    --name "${container}" \
    -p 127.0.0.1::9200 \
    -e "cluster.name=go-cerebro-e2e-${major}" \
    -e "discovery.type=single-node" \
    -e "xpack.security.enabled=false" \
    -e "ES_JAVA_OPTS=-Xms512m -Xmx512m" \
    "${image}" >/dev/null

  host_port="$(docker port "${container}" 9200/tcp | awk -F: '{print $NF}' | tail -n 1)"
  es_url="http://127.0.0.1:${host_port}"
  wait_for_elasticsearch "${es_url}"

  CEREBRO_E2E_ES_URL="${es_url}" CEREBRO_E2E_ES_MAJOR="${major}" go test -tags=e2e ./internal/e2e -count=1 -v

  docker rm -f "${container}" >/dev/null
  container=""
done
