# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=indiekku
BINARY_PATH=./cmd/indiekku
MATCH_BINARY_NAME=indiekku-match
MATCH_BINARY_PATH=./cmd/indiekku-match
BIN_DIR=bin

# Version from git tag, fallback to "dev"
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X main.version=$(VERSION)

.PHONY: all build clean test deps tidy run build-test build-match

all: build

build:
	@echo "Updating embedded web UI..."
	@cp web/index.html internal/api/webui_index.html
	@cp web/history.html internal/api/webui_history.html
	@cp web/logs.html internal/api/webui_logs.html
	@cp web/deploy.html internal/api/webui_deploy.html
	@cp web/styles.css internal/api/webui_styles.css
	@cp web/favicon.svg internal/api/webui_favicon.svg
	@echo "Updating embedded Dockerfile..."
	@cp Dockerfile internal/docker/dockerfile_embed
	mkdir -p $(BIN_DIR)
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) $(BINARY_PATH)
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(MATCH_BINARY_NAME) $(MATCH_BINARY_PATH)

build-match:
	mkdir -p $(BIN_DIR)
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(MATCH_BINARY_NAME) $(MATCH_BINARY_PATH)

clean:
	@echo "Shutting down indiekku server..."
	-./$(BIN_DIR)/$(BINARY_NAME) shutdown 2>/dev/null || true
	@sleep 1
	@echo "Killing any remaining indiekku processes..."
	-pkill -9 indiekku 2>/dev/null || true
	@echo "Cleaning build artifacts and data..."
	$(GOCLEAN)
	rm -rf $(BIN_DIR)
	rm -rf dockerfiles/
	rm -rf game_server/*
	@# Restore .gitkeep
	@touch game_server/.gitkeep
	rm -f indiekku.db
	rm -f .indiekku_apikey
	rm -f indiekku.pid
	rm -f indiekku.log
	@echo "Cleaned: bin/, dockerfiles/, game_server/*, indiekku.db, .indiekku_apikey, indiekku.pid, indiekku.log"

test:
	$(GOTEST) -v ./...

deps:
	$(GOMOD) download

tidy:
	$(GOMOD) tidy

run: build
	./$(BIN_DIR)/$(BINARY_NAME)

install:
	$(GOBUILD) -o $(GOPATH)/bin/$(BINARY_NAME) $(BINARY_PATH)

build-test: build
	@echo "Shutting down existing server..."
	-./$(BIN_DIR)/$(BINARY_NAME) shutdown 2>/dev/null || true
	@sleep 1
	@echo "Starting server..."
	./$(BIN_DIR)/$(BINARY_NAME) serve
	@echo "Done! Server running. Use web UI or CLI to start game servers."