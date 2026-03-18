BINARY=dq
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "-X github.com/dumbmachine/db-cli/cmd.Version=$(VERSION) -X github.com/dumbmachine/db-cli/cmd.Commit=$(COMMIT) -X github.com/dumbmachine/db-cli/cmd.BuildDate=$(BUILD_DATE)"

.PHONY: build install clean test

build:
	go build $(LDFLAGS) -o $(BINARY) .

install:
	go install $(LDFLAGS) .

clean:
	rm -f $(BINARY)

test:
	go test ./...
