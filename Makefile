BINARY := graphmigrate
BUILD_DIR := ./bin

.PHONY: all build run test clean migrate status rollback unlock help

all: build

## build: compile the binary into ./bin/
build:
	@mkdir -p $(BUILD_DIR)
	@echo "→ building $(BINARY)..."
	go build -o $(BUILD_DIR)/$(BINARY) .
	@echo "✓ built: $(BUILD_DIR)/$(BINARY)"

## run: build and run with no arguments (shows help)
run: build
	$(BUILD_DIR)/$(BINARY)

## migrate: apply all pending migrations
migrate: build
	$(BUILD_DIR)/$(BINARY) migrate

## migrate-dry: preview pending migrations without applying
migrate-dry: build
	$(BUILD_DIR)/$(BINARY) migrate --dry-run

## migrate-verbose: apply migrations with debug output
migrate-verbose: build
	$(BUILD_DIR)/$(BINARY) migrate --verbose

## status: show current migration state
status: build
	$(BUILD_DIR)/$(BINARY) status


## unlock: remove a stale migration lock
unlock: build
	$(BUILD_DIR)/$(BINARY) unlock

## test: run all unit tests
test:
	go test ./... -v

## vet: run go vet
vet:
	go vet ./...

## clean: remove compiled binary
clean:
	@rm -rf $(BUILD_DIR)
	@echo "✓ cleaned"

## help: show this help
help:
	@echo ""
	@echo "GraphMigrate — available make targets:"
	@echo ""
	@grep -E '^## ' Makefile | sed 's/## /  /'
	@echo ""
