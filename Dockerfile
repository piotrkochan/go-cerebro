FROM golang:1.26-alpine AS build
RUN apk add --no-cache build-base
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /out/cerebro ./cmd/cerebro

FROM alpine:3.20
RUN apk add --no-cache ca-certificates sqlite-libs
WORKDIR /opt/cerebro
COPY --from=build /out/cerebro /usr/local/bin/cerebro
COPY --from=build /src/public /opt/cerebro/public
COPY --from=build /src/conf /opt/cerebro/conf

EXPOSE 9000
ENTRYPOINT ["cerebro", "serve"]
