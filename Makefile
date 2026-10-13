BINARY  := terraform-provider-wayfinder
VERSION ?= dev

# Allow `make WAYFINDER_GIT_URL=file:///path/to/wayfinder <target>`.
export WAYFINDER_GIT_URL

.PHONY: build test testacc generate lint docs wayfinder-src manifest manifestgen-test bump-wayfinder

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

wayfinder-src:
	./scripts/wayfinder-checkout.sh

manifest:
	./scripts/manifestgen.sh run

manifestgen-test:
	./scripts/manifestgen.sh test

bump-wayfinder:
	@if [ -z "$(REF)" ]; then echo "usage: make bump-wayfinder REF=vX.Y.Z" >&2; exit 2; fi
	@echo "bump-wayfinder: implemented in bump script" >&2; exit 1
