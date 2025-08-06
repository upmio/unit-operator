PROJECT_NAME := "unit-agent"
MAIN_FILE_PAHT := "main.go"
PKG := "gitlab.bsgchina.com/upm/$(PROJECT_NAME)"

BUILD_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
BUILD_COMMIT := $(shell git rev-parse HEAD)
BUILD_COMMIT_SHORT := $(shell git rev-parse --short HEAD)
BUILD_TIME := ${shell date '+%Y-%m-%d %H:%M:%S'}
BUILD_GO_VERSION := $(shell go version | grep -o  'go[0-9].[0-9].*')
VERSION_PATH := "${PKG}/version"

DOCKER_REPOSITORY="harbor.bsgchina.com"
DOCKER_REGISTRY="library/$(PROJECT_NAME)"
DOCKER_TAG=${BUILD_BRANCH}-${BUILD_COMMIT_SHORT}
DOCKER_IMAGE_NAME="$(DOCKER_REPOSITORY)/$(DOCKER_REGISTRY):$(DOCKER_TAG)"

.PHONY: all dep lint vet test test-coverage build clean

all: build

dep: ## Get the dependencies
	@go mod tidy

lint: ## Lint Golang files
	@golint -set_exit_status ${PKG_LIST}

vet: ## Run go vet
	@go vet ${PKG_LIST}

test: ## Run unittests
	@go test -short ${PKG_LIST}

test-coverage: ## Run tests with coverage
	@go test -short -coverprofile cover.out -covermode=atomic ${PKG_LIST}
	@cat cover.out >> coverage.txt

build: dep ## Build the binary file
	@go build -a -o dist/${OUTPUT_NAME} -ldflags "-s -w" -ldflags "-X '${VERSION_PATH}.GIT_BRANCH=${BUILD_BRANCH}' -X '${VERSION_PATH}.GIT_COMMIT=${BUILD_COMMIT}' -X '${VERSION_PATH}.BUILD_TIME=${BUILD_TIME}' -X '${VERSION_PATH}.GO_VERSION=${BUILD_GO_VERSION}'" ${MAIN_FILE}

linux: dep ## Build the binary file
	@GOOS=linux GOARCH=amd64 go build -a -o dist/${OUTPUT_NAME} -ldflags "-s -w" -ldflags "-X '${VERSION_PATH}.GIT_BRANCH=${BUILD_BRANCH}' -X '${VERSION_PATH}.GIT_COMMIT=${BUILD_COMMIT}' -X '${VERSION_PATH}.BUILD_TIME=${BUILD_TIME}' -X '${VERSION_PATH}.GO_VERSION=${BUILD_GO_VERSION}'" ${MAIN_FILE}

install: dep ## install grpc gen tools
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/favadi/protoc-go-inject-tag@latest

run: # Run Develop server
	@go run $(MAIN_FILE) start

clean: ## Remove previous build
	@rm -f dist/*

push: # push git to multi repo
	@git push -u gitee
	@git push -u origin

image: ## Build docker image
	@docker build -t ${DOCKER_IMAGE_NAME} --build-arg http_proxy="$(printenv http_proxy)" --build-arg https_proxy="$(printenv https_proxy)" .
	@docker push ${DOCKER_IMAGE_NAME}
help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'