
VERSION := $(shell git describe --tags --always --dirty)
GO = go
TEST_FLAGS = -v -count=1 -race
PLUGIN_EXAMPLE = plugins/plugins/example
TAG = v$(patsubst v%,%,$(V))

all: build

build: build-ui build-go

build-ui:
	cd ui && npm install && npm run build

build-go:
	$(GO) build -trimpath -ldflags "-s -w -X github.com/sted/smoothdb/version.Version=$(VERSION)"

# Update dependencies everywhere: main module, plugin example and UI.
# Go modules get minor/patch updates plus the latest Go patch release
# (keeps host and plugin toolchains in sync); majors are manual.
# The UI gets npm updates within the semver ranges in package.json.
update: update-go update-ui

update-go:
	$(GO) get go@patch && $(GO) get -u ./... && $(GO) mod tidy
	cd $(PLUGIN_EXAMPLE) && $(GO) get go@patch && $(GO) get -u ./... && $(GO) mod tidy
	$(GO) run golang.org/x/vuln/cmd/govulncheck@latest ./...

update-ui:
	cd ui && npm update --save

test:
	$(GO) test $(TEST_FLAGS) ./database
	$(GO) test $(TEST_FLAGS) ./test/api
	$(GO) test $(TEST_FLAGS) ./test/postgrest
	$(GO) test $(TEST_FLAGS) ./test/recursive

# This is used to recreate PostgREST fixtures
prepare-postgrest-tests:
	psql -U postgres -c "drop database if exists pgrest" && \
		psql -U postgres -c "create database pgrest" && \
		psql -U postgres -f ./test/postgrest/fixtures/load.sql pgrest

# Tag a release and push it to GitHub, where the goreleaser workflow
# builds and publishes the bundles.
# Usage: make release V=0.7.0
release:
	@[ -n "$(V)" ] || { echo "usage: make release V=x.y.z"; exit 1; }
	@git diff-index --quiet HEAD -- || { echo "error: working tree not clean"; exit 1; }
	@[ "$$(git branch --show-current)" = "main" ] || { echo "error: not on branch main"; exit 1; }
	git tag -a $(TAG) -m "Release $(TAG)"
	git push origin main $(TAG)

clean:
	$(GO) clean

.PHONY: all build build-ui build-go update update-go update-ui test prepare-postgrest-tests release clean
