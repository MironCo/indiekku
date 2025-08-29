# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=sdd-server
BINARY_PATH=./cmd/sdd-server

.PHONY: all build clean test deps tidy run

all: deps build

build:
	$(GOBUILD) -o $(BINARY_NAME) $(BINARY_PATH)

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

test:
	$(GOTEST) -v ./...

deps:
	$(GOGET) -d ./...

tidy:
	$(GOMOD) tidy

run: build
	./$(BINARY_NAME)

install:
	$(GOBUILD) -o $(GOPATH)/bin/$(BINARY_NAME) $(BINARY_PATH)