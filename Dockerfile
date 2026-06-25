FROM golang:1.26-alpine AS build
RUN apk add --no-cache nodejs npm
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build
RUN CGO_ENABLED=0 go build -trimpath -o /out/cerebro ./cmd/cerebro

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /opt/cerebro
COPY --from=build /out/cerebro /usr/local/bin/cerebro
COPY --from=build /src/conf /opt/cerebro/conf

EXPOSE 9000
ENTRYPOINT ["cerebro", "serve"]
