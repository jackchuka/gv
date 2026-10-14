.PHONY: build test lint install clean

BINARY := gv
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
PKG := github.com/jackchuka/gv/internal/version
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags "-X $(PKG).Version=$(VERSION) -X $(PKG).Commit=$(COMMIT) -X $(PKG).Date=$(DATE)"

build:
	go build $(LDFLAGS) -o $(BINARY) .

test:
	go test ./... -v

lint:
	golangci-lint run

install:
	go install $(LDFLAGS) .

clean:
	rm -f $(BINARY)
