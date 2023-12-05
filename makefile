IGNORE_MOD_DIR := ./cmd/woco

ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort)
GO_MOD_DIRS := $(filter-out $(IGNORE_MOD_DIR), $(ALL_GO_MOD_DIRS))
ALL_COVERAGE_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort)
COVERAGE_MOD_DIRS := $(filter-out $(IGNORE_MOD_DIR), $(ALL_COVERAGE_MOD_DIRS))
COVERAGE_MODE    = atomic
COVERAGE_PROFILE = coverage.out

GO = go
TIMEOUT = 60

# Build
.PHONY: build
build: $(GO_MOD_DIRS:%=build/%)
build/%:
	@echo "$(GO) build ./..." \
		&& cd ./ \
		&& $(GO) build ./...
build/%: DIR=$*
build/%:
	@echo "$(GO) build $(DIR)/..." \
		&& cd $(DIR) \
		&& $(GO) build ./...

# Tests

TEST_TARGETS := test-default test-bench test-short test-verbose test-race

.PHONY: $(TEST_TARGETS) test
test-default test-race: ARGS=-race
test-bench:   ARGS=-run=xxxxxMatchNothingxxxxx -test.benchtime=1ms -bench=
test-short:   ARGS=-short
test-verbose: ARGS=-v
$(TEST_TARGETS): test
test: test-root $(GO_MOD_DIRS:%=test/%)
test-root:
	@echo "$(GO) test -timeout $(TIMEOUT)s $(ARGS) ./..." \
		&& cd ./ \
		&& $(GO) test -timeout $(TIMEOUT)s $(ARGS) ./...
test/%: DIR=$*
test/%:
	@echo "$(GO) test -timeout $(TIMEOUT)s $(ARGS) -covermode=$(COVERAGE_MODE) -coverprofile=$(COVERAGE_PROFILE) $(DIR)/..." \
		&& cd $(DIR) \
		&& $(GO) test -timeout $(TIMEOUT)s $(ARGS) -covermode=$(COVERAGE_MODE) -coverprofile=$(COVERAGE_PROFILE) ./...

.PHONY: test-coverage
test-coverage: $(COVERAGE_MOD_DIRS:%=test-coverage/%)
test-coverage/%:
	@echo "$(GO) test -timeout $(TIMEOUT)s $(ARGS) -covermode=$(COVERAGE_MODE) -coverprofile=$(COVERAGE_PROFILE) ./..." \
		&& cd ./ \
		&& $(GO) test -timeout $(TIMEOUT)s $(ARGS) -covermode=$(COVERAGE_MODE) -coverprofile=$(COVERAGE_PROFILE) ./...
test-coverage/%: DIR=$*
test-coverage/%:
	@echo "$(GO) test -timeout $(TIMEOUT)s $(ARGS) -covermode=$(COVERAGE_MODE) -coverprofile=$(COVERAGE_PROFILE) $(DIR)/..." \
		&& cd $(DIR) \
		&& $(GO) test -timeout $(TIMEOUT)s $(ARGS) -covermode=$(COVERAGE_MODE) -coverprofile=$(COVERAGE_PROFILE) ./...

golangci-lint:
	cd ./ \
	&& golangci-lint run