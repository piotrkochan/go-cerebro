#!/usr/bin/env bash
set -euo pipefail

image="${CEREBRO_E2E_LDAP_IMAGE:-go-cerebro-e2e-openldap:v2.5.0}"
container="go-cerebro-e2e-ldap-${RANDOM}"
tmp_dir="$(mktemp -d)"

cleanup() {
  docker rm -f "${container}" >/dev/null 2>&1 || true
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

wait_for_ldaps() {
  local port="$1"
  local deadline=$((SECONDS + 90))
  until nc -z 127.0.0.1 "${port}" >/dev/null 2>&1; do
    if (( SECONDS >= deadline )); then
      docker logs "${container}" >&2 || true
      echo "LDAP test server did not start on LDAPS port ${port}" >&2
      return 1
    fi
    sleep 1
  done
}

wait_for_cert() {
  local deadline=$((SECONDS + 90))
  until docker exec "${container}" test -s /etc/ldap/ssl/ldap.crt >/dev/null 2>&1; do
    if (( SECONDS >= deadline )); then
      docker logs "${container}" >&2 || true
      echo "LDAP test server did not create /etc/ldap/ssl/ldap.crt" >&2
      return 1
    fi
    sleep 1
  done
}

wait_for_directory() {
  local deadline=$((SECONDS + 90))
  until docker exec "${container}" ldapsearch -x -H ldapi:/// -D cn=admin,dc=planetexpress,dc=com -w GoodNewsEveryone -b dc=planetexpress,dc=com cn=admin_staff >/dev/null 2>&1; do
    if (( SECONDS >= deadline )); then
      docker logs "${container}" >&2 || true
      echo "LDAP test directory was not populated" >&2
      return 1
    fi
    sleep 1
  done
}

create_test_cert() {
  openssl req \
    -x509 \
    -newkey rsa:2048 \
    -nodes \
    -days 1 \
    -subj "/CN=localhost" \
    -addext "subjectAltName=DNS:localhost,IP:127.0.0.1" \
    -keyout "${tmp_dir}/ldap.key" \
    -out "${tmp_dir}/ldap.crt" >/dev/null 2>&1
  chmod 0644 "${tmp_dir}/ldap.key" "${tmp_dir}/ldap.crt"
}

if [[ "${CEREBRO_E2E_LDAP_IMAGE:-}" == "" ]]; then
  docker build -q -t "${image}" "https://github.com/rroemhild/docker-test-openldap.git#v2.5.0" >/dev/null
fi

create_test_cert

docker rm -f "${container}" >/dev/null 2>&1 || true
docker run -d \
  --name "${container}" \
  -p 127.0.0.1::10636 \
  -v "${tmp_dir}/ldap.crt:/etc/ldap/ssl/ldap.crt:ro" \
  -v "${tmp_dir}/ldap.key:/etc/ldap/ssl/ldap.key:ro" \
  "${image}" >/dev/null

ldaps_port="$(docker port "${container}" 10636/tcp | awk -F: '{print $NF}' | tail -n 1)"
wait_for_ldaps "${ldaps_port}"
wait_for_cert
wait_for_directory

CEREBRO_E2E_LDAP_URL="ldaps://localhost:${ldaps_port}" \
CEREBRO_E2E_LDAP_CA_CERT_FILE="${tmp_dir}/ldap.crt" \
  go test -tags=e2e ./internal/e2e -run TestLDAPRBACGroupsOverLDAPS -count=1 -v
