# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=indiekku
BINARY_PATH=./cmd/indiekku
BIN_DIR=bin

# Version from git tag, fallback to "dev"
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X main.version=$(VERSION)

.PHONY: all build clean test deps tidy run build-test

all: build

build:
	mkdir -p $(BIN_DIR)
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) $(BINARY_PATH)

clean:
	$(GOCLEAN)
	rm -rf $(BIN_DIR)

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
	@sleep 2
	@echo "Starting game server..."
	./$(BIN_DIR)/$(BINARY_NAME) start
	@echo "Done! Check status with: ./bin/indiekku ps"