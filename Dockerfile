# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM node:24-alpine AS frontend
WORKDIR /src
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY . .
RUN npm run build:vite

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
COPY --from=frontend /src/internal/server/static ./internal/server/static
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    target_os="${TARGETOS:-$(go env GOOS)}"; \
    target_arch="${TARGETARCH:-$(go env GOARCH)}"; \
    goarm=""; \
    case "${target_arch}/${TARGETVARIANT:-}" in \
      arm/v5) goarm="5" ;; \
      arm/v6) goarm="6" ;; \
      arm/v7) goarm="7" ;; \
    esac; \
    if [ -n "${goarm}" ]; then \
      CGO_ENABLED=0 GOOS="${target_os}" GOARCH="${target_arch}" GOARM="${goarm}" go build -trimpath -o /out/cerebro ./cmd/cerebro; \
    else \
      CGO_ENABLED=0 GOOS="${target_os}" GOARCH="${target_arch}" go build -trimpath -o /out/cerebro ./cmd/cerebro; \
    fi

FROM --platform=$BUILDPLATFORM alpine:3.24 AS certs
RUN apk add --no-cache ca-certificates

FROM alpine:3.24
WORKDIR /opt/cerebro
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/cerebro /usr/local/bin/cerebro
COPY --from=build /src/conf /opt/cerebro/conf

EXPOSE 9000
ENTRYPOINT ["cerebro", "serve"]
