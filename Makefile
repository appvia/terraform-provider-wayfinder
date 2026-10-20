# Copyright 2026 Appvia Ltd.
# SPDX-License-Identifier: Apache-2.0

BINARY  := terraform-provider-wayfinder
VERSION ?= dev

# Allow `make WAYFINDER_GIT_URL=file:///path/to/wayfinder <target>`.
export WAYFINDER_GIT_URL

.PHONY: build test testacc generate lint license-headers check-license-headers docs wayfinder-src manifest manifestgen-test bump-wayfinder test-scripts check-reproducible

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
	./scripts/license-headers.sh --check

license-headers:
	./scripts/license-headers.sh

check-license-headers:
	./scripts/license-headers.sh --check

TFPLUGINDOCS = go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

# Needs the terraform CLI (>= 1.11) on PATH.
docs:
	$(TFPLUGINDOCS) generate --provider-name wayfinder --rendered-provider-name Wayfinder
	$(TFPLUGINDOCS) validate --provider-name wayfinder
	./scripts/check-docs.sh docs

wayfinder-src:
	./scripts/wayfinder-checkout.sh

manifest:
	./scripts/manifestgen.sh run

manifestgen-test:
	./scripts/manifestgen.sh test

bump-wayfinder:
	@if [ -z "$(REF)" ]; then echo "usage: make bump-wayfinder REF=vX.Y.Z" >&2; exit 2; fi
	./scripts/bump-wayfinder.sh "$(REF)" $(BUMP_ARGS)

test-scripts:
	./scripts/test-ci-quiet.sh
	./scripts/test-wayfinder-checkout.sh
	./scripts/test-bump-wayfinder.sh
	./scripts/test-license-headers.sh

check-reproducible:
	./scripts/check-reproducible.sh
