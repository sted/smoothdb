
VERSION=$(shell git describe --tags `git rev-list --tags --max-count=1`)
GO=go
TEST_FLAGS=-v -count=1 -race

all: build

build: build-ui build-go

build-ui:
	cd ui && npm install && npm run build

build-go:
	$(GO) build -trimpath -ldflags "-s -w -X github.com/sted/smoothdb/version.Version=$(VERSION)"

test:
	$(GO) test $(TEST_FLAGS) ./database
	$(GO) test $(TEST_FLAGS) ./test/api
	$(GO) test $(TEST_FLAGS) ./test/postgrest

# This is used to recreate PostgREST fixtures
prepare-postgrest-tests:
	psql -U postgres -c "drop database if exists pgrest" && \
		psql -U postgres -c "create database pgrest" && \
		psql -U postgres -f ./test/postgrest/fixtures/load.sql pgrest

clean:
	go clean

.PHONY: all build build-ui build-go clean test
