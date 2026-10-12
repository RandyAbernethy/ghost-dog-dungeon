GO ?= go
BIN_DIR ?= bin

.PHONY: build run all linux windows macos test vet fmt check

build:
	$(GO) build -o "$(BIN_DIR)/ghostdog" ./src

run: build
	"$(BIN_DIR)/ghostdog"

all: linux windows macos

linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -o "$(BIN_DIR)/ghostdog-linux-amd64" ./src
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -o "$(BIN_DIR)/ghostdog-linux-arm64" ./src

windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build -o "$(BIN_DIR)/ghostdog-windows-amd64.exe" ./src
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 $(GO) build -o "$(BIN_DIR)/ghostdog-windows-arm64.exe" ./src

macos:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build -o "$(BIN_DIR)/ghostdog-darwin-amd64" ./src
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build -o "$(BIN_DIR)/ghostdog-darwin-arm64" ./src

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

check: test vet
	git diff --check
