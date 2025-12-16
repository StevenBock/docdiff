.PHONY: build install clean test lint fmt

BINARY_NAME=docdiff
BUILD_DIR=bin
VERSION?=0.1.0
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/docdiff

install:
	go install $(LDFLAGS) ./cmd/docdiff

clean:
	rm -rf $(BUILD_DIR)
	go clean

test:
	go test -v ./...

lint:
	golangci-lint run

fmt:
	go fmt ./...

# Build for multiple platforms
build-all: clean
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/docdiff
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/docdiff
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/docdiff
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/docdiff
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/docdiff

# Run the CLI
run:
	go run ./cmd/docdiff $(ARGS)

# Development: build and run
dev: build
	./$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)
