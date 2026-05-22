# Makefile for Kashvi Framework Development

.PHONY: all install build run test tidy fmt vet coverage check-docs clean help

all: help

## install: Install the kashvi CLI binary to $GOPATH/bin
install:
	@echo "Installing kashvi CLI..."
	go install ./cmd/kashvi/

## build: Build the local binaries (cli and server) to ./bin
build:
	@echo "Building binaries..."
	@mkdir -p bin
	go build -o bin/kashvi ./cmd/kashvi
	go build -o bin/server ./cmd/server

## run: Run the framework's fallback server
run:
	@echo "Starting fallback server..."
	go run ./cmd/server

## test: Run unit tests
test:
	@echo "Running tests..."
	go test ./...

## tidy: Run go mod tidy
tidy:
	@echo "Tidying module dependencies..."
	go mod tidy

## fmt: Format all Go files
fmt:
	@echo "Formatting Go files..."
	go fmt ./...

## vet: Run go vet
vet:
	@echo "Vetting Go files..."
	go vet ./...

## coverage: Run the test coverage script
coverage:
	@echo "Checking test coverage..."
	python3 check_coverage.py

## check-docs: Check if all exported symbols are documented in README.md
check-docs:
	@echo "Checking exported symbols documentation..."
	python3 check_docs.py

## clean: Clean up build artifacts and local databases
clean:
	@echo "Cleaning up..."
	rm -rf bin/
	rm -f kashvi.db
	rm -f coverage.out

## help: Show this help message
help:
	@echo "Kashvi Framework Makefile"
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'