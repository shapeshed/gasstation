MODULE   := $(shell go list -m)
DATE     := $(shell date +%FT%T%z)
VERSION  := $(shell git describe --tags --always --dirty --match=v* 2>/dev/null || echo v0)
BIN      := bin
TARGET   := gasstation
SRC      := ./cmd/$(TARGET)/main.go
GO       := go
GOBIN    := $(shell $(GO) env GOPATH)/bin

.PHONY: all
all: lint build

.PHONY: build
build: | $(BIN)
	@echo "Building $(TARGET)..."
	@$(GO) build -v -mod=readonly -tags release \
		-ldflags "-X main.Version=$(VERSION) -X main.BuildDate=$(DATE)" \
		-o $(BIN)/$(TARGET) $(SRC)

.PHONY: lint
lint:
	@echo "Running golangci-lint..."
	@$(GO) tool golangci-lint run ./...

.PHONY: fix
fix: 
	@echo "Running golangci-lint fix..."
	@$(GO) tool golangci-lint run --fix ./...

.PHONY: test
test: 
	@echo "Running tests..."
	@$(GO) test ./...

.PHONY: test-cover
test-cover:
	@echo "Running tests with coverage..."
	@$(GO) test -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html

.PHONY: generate-mocks
generate-mocks:
	@echo "Running mocks..."
	@$(GO) tool mockery 

.PHONY: clean
clean:  
	@echo "Cleaning up..."
	@rm -rf $(BIN) coverage.out coverage.html 

.PHONY: version
version:  
	@echo $(VERSION)

.PHONY: help
help: 
	@echo "Available commands:"
	@echo "  all            Build all client binaries"
	@echo "  lint           Run golangci-lint"
	@echo "  fix            Auto-fix linting issues with golangci-lint --fix"
	@echo "  generate-mocks Generate mocks using mockery"
	@echo "  test           Run all tests"
	@echo "  test-cover     Run tests with coverage and generate a report"
	@echo "  clean          Remove build artifacts"
	@echo "  version        Show project version"
	@echo "  help           Show this help message"
