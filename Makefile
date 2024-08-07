# Makefile for ssz project

default: all

all: setup build test

build: check_consensus_tests
	@echo "Building project..."
	@go build -v ./...

test: check_consensus_tests
	@echo "Running tests..."
	@go test ./...

tidy:
	@echo "Tidying go modules..."
	@go mod tidy

generate:
	@echo "Generating code..."
	@go generate ./...

setup:
	@mkdir -p coverage
	@echo "Downloading consensus tests... This may take a while due to the large repository size."
	@git submodule update --init --recursive --depth=1
	@echo "Consensus tests download completed."

check_consensus_tests:
	@if [ -z "$(shell ls -A tests/testdata/consensus-spec-tests)" ]; then \
		echo "Consensus spec tests directory is empty. Running setup..."; \
		$(MAKE) setup; \
	fi

# Phony targets
.PHONY: all build test tidy generate setup check_consensus_tests
