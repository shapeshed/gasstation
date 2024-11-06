MODULE   = $(shell $(GO) list -m)
DATE    ?= $(shell date +%FT%T%z)
VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || \
			cat .version 2> /dev/null || echo v0)
PKGS     = $(or $(PKG),$(shell $(GO) list ./...))
BIN      = bin

GO      = go
TIMEOUT = 15
V = 0
Q = $(if $(filter 1,$V),,@)
M = $(shell if [ "$$(tput colors 2> /dev/null || echo 0)" -ge 8 ]; then printf "\033[34;1m▶\033[0m"; else printf "▶"; fi)

GENERATED =

CLIENT_DIRS := $(wildcard cmd/*)
EXCLUDED_BINARIES = goimports revive
CLIENT_BINARIES := $(filter-out $(EXCLUDED_BINARIES), $(notdir $(CLIENT_DIRS)))


# Build all client binaries
.PHONY: all
all: fmt lint $(CLIENT_BINARIES)

$(CLIENT_BINARIES): % : | $(BIN)
	$(info $(M) Building executable: $@)
	$Q $(GO) build \
		-tags release \
		-ldflags '-X main.Version=$(VERSION) -X main.BuildDate=$(DATE)' \
		-o $(BIN)/$@ cmd/$@/main.go

###############################################################################
###                                  Tools                                  ###
###############################################################################

$(BIN):
	@mkdir -p $@
$(BIN)/%: | $(BIN) ; $(info $(M) building $(PACKAGE)…)
	$Q env GOBIN=$(abspath $(BIN)) $(GO) install $(PACKAGE)

GOIMPORTS = $(BIN)/goimports
$(BIN)/goimports: PACKAGE=golang.org/x/tools/cmd/goimports@latest

REVIVE = $(BIN)/revive
$(BIN)/revive: PACKAGE=github.com/mgechev/revive@v1.2.4

GOCOV = $(BIN)/gocov
$(BIN)/gocov: PACKAGE=github.com/axw/gocov/gocov@latest

GOCOVXML = $(BIN)/gocov-xml
$(BIN)/gocov-xml: PACKAGE=github.com/AlekSi/gocov-xml@latest

GOTESTSUM = $(BIN)/gotestsum
$(BIN)/gotestsum: PACKAGE=gotest.tools/gotestsum@latest

GOLANGCI-LINT = $(BIN)/golangci-lint
$(BIN)/golangci-lint: PACKAGE=github.com/golangci/golangci-lint/cmd/golangci-lint@latest

###############################################################################
###                                  Tests                                  ###
###############################################################################

TEST_TARGETS := test-short test-race
.PHONY: $(TEST_TARGETS) check test tests
test-short:   ARGS=-short        ## Run only short tests
test-race:    ARGS=-race         ## Run tests with race detector
$(TEST_TARGETS): NAME=$(MAKECMDGOALS:test-%=%)
$(TEST_TARGETS): test
check test tests: fmt lint $(GENERATED) | $(GOTESTSUM) ; $(info $(M) running $(NAME:%=% )tests…) @ ## Run tests
	$Q mkdir -p test
	$Q $(GOTESTSUM) --junitfile test/tests.xml -- -timeout $(TIMEOUT)s $(ARGS) $(PKGS)
.PHONY: test-bench
test-bench: $(GENERATED) ; $(info $(M) running benchmarks…) @ ## Run benchmarks
	$Q $(GOTESTSUM) -f standard-quiet -- --timeout $(TIMEOUT)s -run=__absolutelynothing__ -bench=. $(PKGS)

COVERAGE_MODE = atomic
.PHONY: test-coverage
test-coverage: fmt lint $(GENERATED)
test-coverage: | $(GOCOV) $(GOCOVXML) $(GOTESTSUM) ; $(info $(M) running coverage tests…) @ ## Run coverage tests
	$Q mkdir -p test
	$Q $(GOTESTSUM) -- \
		-coverpkg=$(shell echo $(PKGS) | tr ' ' ',') \
		-covermode=$(COVERAGE_MODE) \
		-coverprofile=test/profile.out $(PKGS)
	$Q $(GO) tool cover -html=test/profile.out -o test/coverage.html
	$Q $(GOCOV) convert test/profile.out | $(GOCOVXML) > test/coverage.xml
	@echo -n "Code coverage: "; \
		echo "scale=1;$$(sed -En 's/^<coverage line-rate="([0-9.]+)".*/\1/p' test/coverage.xml) * 100 / 1" | bc -q

.PHONY: lint
 # lint: | $(REVIVE) ; $(info $(M) running golint…) @ ## Run golint
#	$Q $(REVIVE) -formatter friendly -set_exit_status ./...
lint: | $(GOLANGCI-LINT) ; $(info $(M) running golint…) @ ## Run golint# 
	$Q $(GOLANGCI-LINT) run

# .PHONY: fmt
# fmt: | $(GOIMPORTS) ; $(info $(M) running gofmt…) @ ## Run gofmt on all source files
# 	$Q $(GOIMPORTS) -local $(MODULE) -w $(shell $(GO) list -f '{{$$d := .Dir}}{{range $$f := .GoFiles}}{{printf "%s/%s\n" $$d $$f}}{{end}}{{range $$f := .CgoFiles}}{{printf "%s/%s\n" $$d $$f}}{{end}}{{range $$f := .TestGoFiles}}{{printf "%s/%s\n" $$d $$f}}{{end}}' $(PKGS))

.PHONY: fmt
fmt: ; $(info $(M) running gofmt…) @ ## Run gofmt on all source files
	$Q find . -name '*.go' -exec gofmt -s -w {} +

# Misc

.PHONY: clean
clean: ; $(info $(M) cleaning…)	@ ## Cleanup everything
	@rm -rf $(BIN) test $(GENERATED)

.PHONY: help
help:
	@grep -hE '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-17s\033[0m %s\n", $$1, $$2}'

.PHONY: version
version:
	@echo $(VERSION)
