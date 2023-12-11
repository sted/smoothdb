
VERSION=$(shell git describe --tags `git rev-list --tags --max-count=1`)
GO=go
TEST_FLAGS=-v -count=1

all: build

build:
	$(GO) build -ldflags "-X main.Version=$(VERSION)"

test:
	$(GO) test $(TEST_FLAGS) ./database
	$(GO) test $(TEST_FLAGS) ./test/api
	$(GO) test $(TEST_FLAGS) ./test/postgrest

# This is used to recreate PostgREST fixtures
prepare-postgrest-tests:
	psql -U postgres -c "drop database if exists pgrest" && \
		psql -U postgres -c "create database pgrest" && \
		psql -U postgres -f ./test/postgrest/fixtures/load.sql pgrest

.PHONY: all build test
