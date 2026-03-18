GOCACHE ?= $(CURDIR)/.gocache
GOMODCACHE ?= $(CURDIR)/.gomodcache

DOCKER ?= docker
IMAGE ?= ilovelili/ad-engine
TAG ?= dev

.PHONY: server-run build fmt tidy docker_build docker_run docker_push

server-run:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go run ./cmd/server

build:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build ./...

fmt:
	gofmt -w $$(find cmd internal -name '*.go' -type f)

tidy:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go mod tidy

docker_build:
	$(DOCKER) build -t $(IMAGE):$(TAG) .

docker_run:
	$(DOCKER) run --rm -p 8080:8080 -e HTTP_ADDR=:8080 -v $(CURDIR)/adengine.db:/data/adengine.db $(IMAGE):$(TAG)

docker_push:
	$(DOCKER) push $(IMAGE):$(TAG)
