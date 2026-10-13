BINARY  := terraform-provider-wayfinder
VERSION ?= dev

.PHONY: build test testacc generate lint docs

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY) .

test:
	go test ./... -count=1

testacc:
	TF_ACC=1 go test ./... -count=1 -timeout 120m

generate:
	go generate ./...

lint:
	golangci-lint run

# Placeholder until generated docs land.
docs:
	@echo "docs: not yet implemented"
