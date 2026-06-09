# ShiroutoCode — build & test
BINARY := shiroutocode
PKG := ./cmd/shiroutocode
BIN_DIR := bin

.PHONY: all build test test-race cover vet fmt vuln clean install cross

all: fmt vet test build

build:
	go build -o $(BIN_DIR)/$(BINARY) $(PKG)

install:
	go install $(PKG)

test:
	go test ./... -count=1

test-race:
	go test ./... -count=1 -race

cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -1

vet:
	go vet ./...

fmt:
	gofmt -l -w .

vuln:
	govulncheck ./...

clean:
	rm -rf $(BIN_DIR) coverage.out

# Cross-compile single static binaries for common platforms.
cross:
	GOOS=darwin  GOARCH=arm64 go build -o $(BIN_DIR)/$(BINARY)-darwin-arm64 $(PKG)
	GOOS=darwin  GOARCH=amd64 go build -o $(BIN_DIR)/$(BINARY)-darwin-amd64 $(PKG)
	GOOS=linux   GOARCH=amd64 go build -o $(BIN_DIR)/$(BINARY)-linux-amd64 $(PKG)
	GOOS=linux   GOARCH=arm64 go build -o $(BIN_DIR)/$(BINARY)-linux-arm64 $(PKG)
