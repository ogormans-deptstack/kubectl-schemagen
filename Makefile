BINARY_NAME := kubectl-schemagen
BUILD_DIR := bin
GO_MODULE := github.com/ogormans-deptstack/kubectl-schemagen
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(BUILD_DATE)"

.PHONY: build test test-unit test-e2e test-coverage lint clean install help

help:
	@echo "Targets:"
	@echo "  build       Build the kubectl-schemagen binary"
	@echo "  test        Run unit tests"
	@echo "  test-unit   Run unit tests with race detection"
	@echo "  test-e2e    Run end-to-end tests (requires kind cluster)"
	@echo "  lint        Run golangci-lint"
	@echo "  clean       Remove build artifacts"
	@echo "  install     Build and install to GOPATH/bin"

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/kubectl-schemagen

test: test-unit

test-unit:
	go test -race -count=1 ./pkg/... ./cmd/... ./internal/...

test-e2e:
	go test -race -count=1 -tags=e2e -timeout=10m ./test/e2e/...

test-coverage:
	go test -race -coverprofile=coverage.out ./pkg/... ./cmd/... ./internal/...
	go tool cover -func=coverage.out
	@echo ""
	@echo "HTML report: go tool cover -html=coverage.out"

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR) dist

install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) $(shell go env GOPATH)/bin/$(BINARY_NAME)
