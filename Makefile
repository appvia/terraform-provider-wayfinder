SHELL = /bin/sh -e
AUTHOR_EMAIL=info@appvia.io
PROJECT=terraform-provider-wayfinder
REGISTRY=ghcr.io
REGISTRY_ORG=appvia
BUILD_TIME=$(shell date '+%s')
CLI_PLATFORMS=darwin linux windows
CLI_ARCHITECTURES=amd64 arm64
CURRENT_TAG=$(shell git tag --points-at HEAD)
DEPS=$(shell go list -f '{{range .TestImports}}{{.}} {{end}}' ./...)
GIT_REMOTE?=origin
GIT_BRANCH?=
GOVERSION ?= 1.19
HARDWARE=$(shell uname -m)
PLATFORM=$(shell uname -s)
PACKAGES_LIST=$(shell go list ./... | sed -e 's/github.com\/appvia\/terraform-provider-wayfinder\//.\//')
PACKAGES=$(shell go list ./...)
GIT_SHA=$(shell git --no-pager describe --always --dirty)
BUILD_TIME=$(shell date '+%s')
GO_DIRS=cmd hack pkg
SH_DIRS=.circleci hack
ROOT_DIR=${PWD}
UNAME := $(shell uname)
LFLAGS ?=
VERSION ?= latest

CLI_PLATFORMS=darwin linux windows
CLI_ARCHITECTURES=amd64 arm64
export GOFLAGS = -mod=vendor

.PHONY: test build release check golangci-lint

default: build

golang:
	@echo "--> Go Version"
	@go version
	@echo "GOFLAGS: $$GOFLAGS"
	@mkdir -p bin

### BUILD ###

build: golang
	@echo "--> Building Terraform Provider"
	@go build -o bin/terraform-provider-wayfinder *.go

### TESTING ###

test:
	@echo "--> Running the tests"
	@rm -f cover.out || true
	@go run ./vendor/gotest.tools/gotestsum/main.go --format pkgname --packages $(PACKAGES_LIST) -- -coverprofile=cover.out
	@echo "--> Coverage: $(shell go tool cover -func=cover.out | grep total | grep -Eo '[0-9]+\.[0-9]+')" || true

### CHECKING AND LINTING ###

check-gofmt:
	@echo "--> Running gofmt check"
	@if ./hack/gofmt.sh -l ${GO_DIRS} | grep \.go ; then \
		echo "There are unformatted Go files - did you forget to run 'make gofmt'?"; \
		exit 1; \
	fi

check-schema:
	@echo "--> Running schema check"
	@cd tests/e2e/schema && terraform init >/dev/null && terraform providers schema -json >/dev/null

gofmt:
	@echo "--> Running go fmt"
	@go fmt $(shell go list ./... | grep -v /vendor/)

format: gofmt

bench:
	@echo "--> Running go bench"
	@go test -bench=. -benchmem

coverage:
	@echo "--> Running go coverage"
	@go test -coverprofile cover.out
	@go tool cover -html=cover.out -o cover.html

spelling:
	@echo "--> Checking the spelling."
	@find . -name "*.go" -type f -not -path "./vendor/*" -not -path "./charts/*" | xargs go run github.com/client9/misspell/cmd/misspell -error -source=go *.go
	@find . -name "*.md" -type f -not -path "./vendor/*" -not -path "./charts/*" | xargs go run github.com/client9/misspell/cmd/misspell -error -source=text *.md

lint:
	@echo "--> Checking against the golangci-lint"
	@go run github.com/golangci/golangci-lint/cmd/golangci-lint run --timeout 5m ./...

shfmt:
	@echo "--> Running shfmt"
	@go run mvdan.cc/sh/v3/cmd/shfmt -l -w -ci -i 2 -- images/assets

check: test
	@echo "--> Running code checkers"
	@$(MAKE) golang
	@$(MAKE) check-gofmt
	@$(MAKE) shfmt
	@$(MAKE) spelling
	@$(MAKE) golangci-lint

### UTILITIES ###

clean:
	@echo "--> Cleaning up the environment"
	rm -rf ./bin 2>/dev/null
	rm -rf ./release 2>/dev/null
