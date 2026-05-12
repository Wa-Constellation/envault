VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build install test lint clean

build:
	go build $(LDFLAGS) -o envault .

install:
	go install $(LDFLAGS) .

test:
	go test ./...

test-v:
	go test -v ./...

lint:
	golangci-lint run

clean:
	rm -f envault

release:
	goreleaser release --clean
