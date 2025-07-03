.PHONY: build clean install test

# Get version from git tag or use dev
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || echo "dev")

# Build the binary
build:
	go build -ldflags "-X main.version=$(VERSION)" -o kubectl-nuke ./cmd/kubectl-nuke

# Install to /usr/local/bin
install: build
	sudo mv kubectl-nuke /usr/local/bin/

# Clean build artifacts
clean:
	rm -f kubectl-nuke

# Run tests
test:
	go test -v ./...

# Build for all platforms (similar to GitHub Actions)
build-all:
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o kubectl-nuke-linux-amd64 ./cmd/kubectl-nuke
	GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o kubectl-nuke-linux-arm64 ./cmd/kubectl-nuke
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o kubectl-nuke-darwin-amd64 ./cmd/kubectl-nuke
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o kubectl-nuke-darwin-arm64 ./cmd/kubectl-nuke
	GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o kubectl-nuke-windows-amd64.exe ./cmd/kubectl-nuke
	GOOS=windows GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o kubectl-nuke-windows-arm64.exe ./cmd/kubectl-nuke

# Show current version that would be used
version:
	@echo $(VERSION)
