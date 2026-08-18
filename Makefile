GO ?= go
BIN_DIR := bin
BINARY := $(BIN_DIR)/ssh-site

# The private headshot the portrait is rendered from. It is not in this repo -
# it is source material, and the matte in art/lib/master.py is traced for that
# one photograph. Override it if it lives somewhere else:
#   make art ART_HEADSHOT=/path/to/headshot.jpg
ART_HEADSHOT ?= .scratch/ssh-site/assets/headshot.jpg

.PHONY: content
content:
	sh scripts/fetch-pack.sh

# Regenerates the checked-in terminal art. A developer tool, never a CI step:
# the art is a build-time asset, so a content-pack push rebuilds the site
# without re-rendering a photograph. Needs figlet, chafa and ImageMagick 7.
.PHONY: art
art:
	python3 art/build.py --headshot "$(ART_HEADSHOT)"

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
