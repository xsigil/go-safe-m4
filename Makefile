BINARY_NAME=safe-m4
CMD_PKG=./cmd/safe-m4
BIN_DIR=bin

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0")
LDFLAGS = -s -w -X main.version=$(VERSION)

.PHONY: all build test clean lint install

all: test build

build:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_PKG)

test:
	go test -v -race ./...

lint:
	go vet ./...

clean:
	rm -rf $(BIN_DIR)

install: build
	install -Dm755 $(BIN_DIR)/$(BINARY_NAME) $(DESTDIR)/usr/bin/$(BINARY_NAME)
