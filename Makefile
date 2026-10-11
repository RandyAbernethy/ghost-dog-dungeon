GO ?= go

.PHONY: build run test vet fmt check

build:
	$(GO) build -o bin/ghostdog ./src

run: build
	./bin/ghostdog

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

check: test vet
	git diff --check
