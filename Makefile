GO ?= go
BIN_DIR := bin
BINARY := $(BIN_DIR)/ssh-site

.PHONY: content
content:
	sh scripts/fetch-pack.sh

.PHONY: build
build:
	$(GO) build -o $(BINARY) ./cmd/ssh-site

.PHONY: run
run: build
	./$(BINARY)

.PHONY: test
test:
	$(GO) test ./...

.PHONY: fmt
fmt:
	@unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt needs to be run on:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

.PHONY: fmt-fix
fmt-fix:
	gofmt -w .

.PHONY: lint
lint:
	golangci-lint run

.PHONY: ci
ci: content fmt vet lint build test

.PHONY: vet
vet:
	$(GO) vet ./...
