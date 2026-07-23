BINARY  := agentrepl
BIN_DIR := bin
PREFIX  ?= $(HOME)/.local

.PHONY: build fmt test install clean

build: $(BIN_DIR)/$(BINARY)

$(BIN_DIR)/$(BINARY): go.mod go.sum $(shell find cmd internal -name '*.go')
	install -d $(BIN_DIR)
	go build -ldflags "-X main.version=$(shell git describe --tags --always --dirty)" -o $(BIN_DIR)/$(BINARY) ./cmd/agentrepl

fmt:
	go fmt ./...

test:
	go test ./...

install: build
	install -d $(PREFIX)/bin
	install -m 0755 $(BIN_DIR)/$(BINARY) $(PREFIX)/bin/$(BINARY)

clean:
	rm -rf $(BIN_DIR)
	go clean
