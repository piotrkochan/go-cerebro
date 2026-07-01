#!/usr/bin/env bash
set -euo pipefail

floci_image="${CEREBRO_E2E_FLOCI_IMAGE:-floci/floci:1.5.29}"
domain="${CEREBRO_E2E_FLOCI_DOMAIN:-cerebro-e2e-${RANDOM}}"
engine_version="${CEREBRO_E2E_FLOCI_ENGINE_VERSION:-OpenSearch_2.11}"
floci_container="${CEREBRO_E2E_FLOCI_CONTAINER:-go-cerebro-e2e-floci-${RANDOM}}"
opensearch_container="floci-opensearch-${domain}"
config_file=""

cleanup() {
  if [[ "${floci_url:-}" != "" ]]; then
    curl -fsS -X DELETE "${floci_url}/2021-01-01/opensearch/domain/${domain}" >/dev/null 2>&1 || true
  fi
  docker rm -f "${opensearch_container}" >/dev/null 2>&1 || true
  docker rm -f "${floci_container}" >/dev/null 2>&1 || true
  if [[ "${config_file}" != "" ]]; then
    rm -f "${config_file}"
  fi
}
trap cleanup EXIT

wait_for_url() {
  local url="$1"
  local label="$2"
  local deadline=$((SECONDS + 240))
  until curl -fsS "${url}" >/dev/null 2>&1; do
    if (( SECONDS >= deadline )); then
      echo "${label} did not become ready at ${url}" >&2
      docker logs "${floci_container}" >&2 || true
      docker logs "${opensearch_container}" >&2 || true
      return 1
    fi
    sleep 2
  done
}

wait_for_container() {
  local name="$1"
  local deadline=$((SECONDS + 240))
  until docker inspect "${name}" >/dev/null 2>&1; do
    if (( SECONDS >= deadline )); then
      echo "container ${name} was not created" >&2
      docker logs "${floci_container}" >&2 || true
      return 1
    fi
    sleep 2
  done
}

docker rm -f "${floci_container}" "${opensearch_container}" >/dev/null 2>&1 || true

docker run -d \
  --name "${floci_container}" \
  -p 127.0.0.1::4566 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -u root \
  -e FLOCI_STORAGE_MODE=memory \
  -e FLOCI_SERVICES_OPENSEARCH_MOCK=false \
  "${floci_image}" >/dev/null

floci_port="$(docker port "${floci_container}" 4566/tcp | awk -F: '{print $NF}' | tail -n 1)"
floci_url="http://127.0.0.1:${floci_port}"

wait_for_url "${floci_url}/_localstack/health" "Floci"

echo "==> Floci OpenSearch ${engine_version}"
curl -fsS --max-time 300 -X POST "${floci_url}/2021-01-01/opensearch/domain" \
  -H "Content-Type: application/json" \
  -d "{
    \"DomainName\":\"${domain}\",
    \"EngineVersion\":\"${engine_version}\",
    \"ClusterConfig\":{\"InstanceType\":\"m5.large.search\",\"InstanceCount\":1},
    \"EBSOptions\":{\"EBSEnabled\":true,\"VolumeType\":\"gp2\",\"VolumeSize\":10}
  }" >/dev/null

wait_for_container "${opensearch_container}"

opensearch_port="$(docker port "${opensearch_container}" 9200/tcp | awk -F: '{print $NF}' | tail -n 1)"
opensearch_url="http://127.0.0.1:${opensearch_port}"

wait_for_url "${opensearch_url}/_cluster/health" "Floci OpenSearch domain"

config_file="$(mktemp)"
cat >"${config_file}" <<YAML
hosts:
  - name: "Floci OpenSearch"
    host: "${opensearch_url}"

auth:
  type: disabled

server:
  secret: "floci-e2e-secret-change-me"

es:
  allow_ad_hoc_hosts: false
  max_response_bytes: 26214400
  aws:
    enabled: true
    region: "${CEREBRO_E2E_AWS_REGION:-us-east-1}"
    service: "${CEREBRO_E2E_AWS_SERVICE:-es}"
    access_key_id: "${CEREBRO_E2E_AWS_ACCESS_KEY_ID:-test}"
    secret_access_key: "${CEREBRO_E2E_AWS_SECRET_ACCESS_KEY:-test}"
YAML

CEREBRO_E2E_CONFIG="${config_file}" \
CEREBRO_E2E_CONFIG_HOST="Floci OpenSearch" \
CEREBRO_E2E_ES_MAJOR=8 \
go test -tags=e2e ./internal/e2e -count=1 -v
