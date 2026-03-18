# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache ca-certificates tzdata build-base

WORKDIR /src

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build a static-ish binary (sqlite uses CGO, so we keep CGO enabled).
# Don't force GOARCH here; it breaks on some non-amd64 builders (e.g. arm64).
RUN CGO_ENABLED=1 GOOS=linux \
    go build -trimpath -ldflags="-s -w" -o /out/ad-engine ./cmd/server


# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /out/ad-engine /usr/local/bin/ad-engine

# Keep the default in sync with README (config can override)
EXPOSE 8080

# Sensible defaults; you can override with -e as needed
ENV HTTP_ADDR=:8080
ENV DB_PATH=/data/adengine.db

# Optional volume for sqlite db
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/ad-engine"]
