# Podplane <https://podplane.dev>
# Copyright The Podplane Authors
# SPDX-License-Identifier: Apache-2.0

.DEFAULT_GOAL := help

BINDIR=bin
REPORTSDIR=reports

BINARY_NAME=seedgen
MAIN_PKG=.

BUILDVARS_PKG=github.com/podplane/seedgen/internal/buildvars

VERSION_TAG=$(shell if git diff --quiet && git diff --cached --quiet; then git describe --tags --exact-match 2>/dev/null; fi)
BUILD_VERSION=$(if $(VERSION_TAG),$(VERSION_TAG),dev)
BUILD_DATE=$(shell date -u '+%Y-%m-%dT%H:%M:%S')
COMMIT_HASH=$(shell git rev-parse --short HEAD)
COMMIT_DATE=$(shell git log -1 --format=%cd --date=format:'%Y-%m-%dT%H:%M:%S')
COMMIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)

# Cross-compilation settings, defaulting OS/ARCH to the current platform.
# seedgen has no CGO dependencies; CGO_ENABLED stays 0 for static binaries on all
# platforms.
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
CGO_ENABLED=0
EXTRA_LD_FLAGS=
ifeq ($(GOOS),linux)
	EXTRA_LD_FLAGS=-extldflags -static
endif

.PHONY: help setup fmt lint precommit test build clean

help: ## Show available targets
	@echo "Usage: make <target>"
	@awk 'BEGIN {FS = ":.*?## "} /^##@/ {printf "\n\033[1m%s\033[0m\n", substr($$0, 5)} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Verify required tools and enable git hooks
	@command -v go >/dev/null 2>&1 || { echo "go is required but not installed"; exit 1; }
	@echo "All required tools are installed."
	@cp scripts/git-hooks/pre-commit .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@cp scripts/git-hooks/commit-msg .git/hooks/commit-msg
	@chmod +x .git/hooks/commit-msg
	@echo "Git hooks installed."

##@ Build & Test

fmt: ## Format Go source files
	@go fmt ./...

lint: ## Run linters
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint is required but not installed"; exit 1; }
	@golangci-lint run

precommit: ## Check formatting and run linters (read-only)
	@echo "Checking formatting..."
	@UNFORMATTED=$$(gofmt -l . 2>&1); \
	if [ -n "$$UNFORMATTED" ]; then \
		echo "The following files need formatting (run 'make fmt'):"; \
		echo "$$UNFORMATTED"; \
		exit 1; \
	fi
	@$(MAKE) lint

test: ## Run tests with race detector
	go test -v -race ./...

build: ## Build the seedgen binary
	mkdir -p $(BINDIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
	go build \
		-o $(BINDIR)/$(BINARY_NAME) \
		-trimpath \
		-ldflags "$(EXTRA_LD_FLAGS) \
		-X $(BUILDVARS_PKG).buildVersion=$(BUILD_VERSION) \
		-X $(BUILDVARS_PKG).buildDate=$(BUILD_DATE) \
		-X $(BUILDVARS_PKG).commitHash=$(COMMIT_HASH) \
		-X $(BUILDVARS_PKG).commitDate=$(COMMIT_DATE) \
		-X $(BUILDVARS_PKG).commitBranch=$(COMMIT_BRANCH) \
		" $(MAIN_PKG)
	printf "%s" "$(BUILD_VERSION)-$(COMMIT_HASH)" > $(BINDIR)/version.txt

clean: ## Remove build artifacts
	rm -rf $(BINDIR)
	rm -rf $(REPORTSDIR)
