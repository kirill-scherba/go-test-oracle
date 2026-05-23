.PHONY: build test lint clean all install

BINARY ?= go-test-oracle
BINARY_MCP ?= go-test-oracle-mcp
GOFLAGS ?= -buildvcs=false

all: build

build:
	go build $(GOFLAGS) -o bin/$(BINARY) ./cmd/go-test-oracle

build-mcp:
	go build $(GOFLAGS) -o bin/$(BINARY_MCP) ./cmd/go-test-oracle-mcp

test:
	go test -v -race ./...

lint:
	gofmt -w .
	go vet ./...

clean:
	rm -rf bin/

install: build
	go install ./cmd/go-test-oracle
