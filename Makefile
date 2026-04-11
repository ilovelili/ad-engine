ifneq (,$(wildcard .env))
include .env
export
endif

GOCACHE ?= $(CURDIR)/.gocache
GOMODCACHE ?= $(CURDIR)/.gomodcache

DOCKER ?= docker
IMAGE ?= ilovelili/ad-engine
TAG ?= dev
HTTP_ADDR ?= :8080
CONNECTION_SECRET ?= dev-local-very-long-random-secret-2026-04-11
META_GRAPH_BASE_URL ?= https://graph.facebook.com
META_GRAPH_API_VERSION ?= v22.0
META_APP_ID ?=
META_APP_SECRET ?=
META_REDIRECT_URI ?= https://adsone.ngrok.io/api/v1/oauth/meta/callback
META_OAUTH_SCOPES ?= ads_management,business_management

.PHONY: server-run test-page ngrok build fmt lint tidy docker_build docker_run docker_push

# Development server
server-run:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go run ./cmd/server

# Local test page with Meta OAuth settings
test-page:
	HTTP_ADDR=$(HTTP_ADDR) \
	CONNECTION_SECRET=$(CONNECTION_SECRET) \
	META_GRAPH_BASE_URL=$(META_GRAPH_BASE_URL) \
	META_GRAPH_API_VERSION=$(META_GRAPH_API_VERSION) \
	META_APP_ID=$(META_APP_ID) \
	META_APP_SECRET=$(META_APP_SECRET) \
	META_REDIRECT_URI=$(META_REDIRECT_URI) \
	META_OAUTH_SCOPES=$(META_OAUTH_SCOPES) \
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go run ./cmd/server

# Start ngrok for a public Meta OAuth redirect URL
ngrok:
	ngrok http --url=adsone.ngrok.io 8080

build:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build ./...

# Format Go source files
fmt:
	gofmt -w $$(find cmd internal -name '*.go' -type f)

# Lint Go code with the standard vet checks
lint:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go vet ./...

# Sync Go module dependencies
tidy:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go mod tidy

# Build the local container image
docker_build:
	$(DOCKER) build -t $(IMAGE):$(TAG) .

# Run the containerized app locally
docker_run:
	$(DOCKER) run --rm -p 8080:8080 -e HTTP_ADDR=:8080 -v $(CURDIR)/adengine.db:/data/adengine.db $(IMAGE):$(TAG)

# Push the container image
docker_push:
	$(DOCKER) push $(IMAGE):$(TAG)
