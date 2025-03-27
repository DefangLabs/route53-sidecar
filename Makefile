# gvm use 1.22
GO_VERSION_REQUIRED = 1.22

# VERSION is the version we should download and use.
VERSION:=$(shell git rev-parse --short HEAD)
# DOCKER is the docker image repo we need to push to.
DOCKER_REPO:=defangio
DOCKER_USER:=defangio
DOCKER_IMAGE_NAME:=$(DOCKER_REPO)/route53-sidecar

DOCKER_IMAGE_ARM64:=$(DOCKER_IMAGE_NAME):arm64-$(VERSION)
DOCKER_IMAGE_AMD64:=$(DOCKER_IMAGE_NAME):amd64-$(VERSION)
BUILD_FLAGS:=-ldflags "-s -w -X main.version=$(VERSION)" -trimpath

.PHONY: check_go_version
check_go_version:
	@if [ $$(go version | cut -d " " -f3 | cut -c 3- | cut -d. -f1-2) != "$(GO_VERSION_REQUIRED)" ]; then \
		echo "Required Go version $(GO_VERSION_REQUIRED) is not installed."; \
		exit 1; \
	fi
	@echo "Using Go version $(GO_VERSION_REQUIRED) for build / test."

.PHONY: help
help:
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/:.*##/:/'

.PHONY: ensure
ensure: check_go_version ## Run go get -u
	go get -t -u ./...

.PHONY: build
build: ensure ## Build a local binary
	go build $(BUILD_FLAGS) -o route53-sidecar

.PHONY: build-amd64
build-amd64: ensure
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o route53-sidecar

.PHONY: build-arm64
build-arm64: ensure
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(BUILD_FLAGS) -o route53-sidecar

.PHONY: docker-amd64
docker-amd64: build-amd64
	docker build --platform linux/amd64 -t route53-sidecar -t $(DOCKER_IMAGE_AMD64) .

.PHONY: docker-arm64
docker-arm64: build-arm64
	docker build --platform linux/arm64 -t route53-sidecar -t $(DOCKER_IMAGE_ARM64) .

.PHONY: docker
docker: docker-amd64 docker-arm64 ## Build all docker images and manifest

.PHONY: push
push: docker login ## Push all docker images
	docker push $(DOCKER_IMAGE_AMD64)
	docker push $(DOCKER_IMAGE_ARM64)
	docker manifest create --amend $(DOCKER_IMAGE_NAME):$(VERSION) $(DOCKER_IMAGE_AMD64) $(DOCKER_IMAGE_ARM64)
	docker manifest push --purge $(DOCKER_IMAGE_NAME):$(VERSION)

.PHONY: push-latest
push-latest: push ## Push all docker images
	docker manifest create --amend $(DOCKER_IMAGE_NAME):latest $(DOCKER_IMAGE_AMD64) $(DOCKER_IMAGE_ARM64)
	docker manifest push --purge $(DOCKER_IMAGE_NAME):latest

.PHONY: push-test
push-test: push ## Push all docker images
	docker manifest create --amend $(DOCKER_IMAGE_NAME):test $(DOCKER_IMAGE_AMD64) $(DOCKER_IMAGE_ARM64)
	docker manifest push --purge $(DOCKER_IMAGE_NAME):test

.PHONY: login
login: ## Login to docker
	@docker login -u $(DOCKER_USER)
