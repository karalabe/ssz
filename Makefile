# Makefile for ssz project

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=ssz

# Git parameters
GITCMD=git

# Build targets
all: test build

test:
	@if [ -z "$(shell ls -A tests/testdata/consensus-spec-tests)" ]; then \
		echo "Consensus spec tests directory is empty. Running setup..."; \
		$(MAKE) setup; \
	fi
	$(GOTEST) -v ./...

tidy:
	$(GOMOD) tidy

generate:
	$(GOCMD) generate ./...

setup:
	@mkdir -p coverage
	@echo "Downloading consensus tests... This may take a while due to the large repository size."
	@$(GITCMD) submodule update --init --recursive --depth=1
	@echo "Consensus tests download completed."

# Phony targets
.PHONY: all build test clean run deps tidy generate coverage submodules
