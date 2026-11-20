# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM node:26-alpine@sha256:6f08a523443ddedf52941874c203475b0ad503e859ad70cb26ab8be9a517dbd5 AS frontend
WORKDIR /src
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY . .
ARG CEREBRO_VERSION
RUN if [ -n "${CEREBRO_VERSION}" ]; then export VITE_APP_VERSION="${CEREBRO_VERSION}"; fi; \
    npm run build:vite

# Go 1.27.4; update with make bumpgo.
FROM --platform=$BUILDPLATFORM golang:1.27.4-alpine@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280 AS build
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
ARG CEREBRO_VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
COPY --from=frontend /src/internal/server/static ./internal/server/static
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    target_os="${TARGETOS:-$(go env GOOS)}"; \
    target_arch="${TARGETARCH:-$(go env GOARCH)}"; \
    ldflags="-X github.com/lmenezes/cerebro/internal/version.Version=${CEREBRO_VERSION}"; \
    goarm=""; \
    case "${target_arch}/${TARGETVARIANT:-}" in \
      arm/v5) goarm="5" ;; \
      arm/v6) goarm="6" ;; \
      arm/v7) goarm="7" ;; \
    esac; \
    if [ -n "${goarm}" ]; then \
      CGO_ENABLED=0 GOOS="${target_os}" GOARCH="${target_arch}" GOARM="${goarm}" go build -trimpath -ldflags="${ldflags}" -o /out/cerebro ./cmd/cerebro; \
    else \
      CGO_ENABLED=0 GOOS="${target_os}" GOARCH="${target_arch}" go build -trimpath -ldflags="${ldflags}" -o /out/cerebro ./cmd/cerebro; \
    fi

FROM --platform=$BUILDPLATFORM alpine:3.24@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf AS certs
RUN apk add --no-cache ca-certificates

FROM alpine:3.24@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf
WORKDIR /opt/cerebro
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/cerebro /usr/local/bin/cerebro
COPY --from=build /src/conf /opt/cerebro/conf

EXPOSE 9000
ENTRYPOINT ["cerebro"]
CMD ["serve", "-config", "/opt/cerebro/conf/application.example.yaml"]
