.PHONY: build build-server build-client build-testbed run-server run-server-lan run-client run-client-local run-testbed run-testbed-onlcr test clean generate-manifest winres

GIT_COMMIT  := $(shell git rev-parse --short HEAD)
BUILD_DATE  := $(shell git log -1 --format=%cI)
GIT_REMOTE  := $(shell git remote get-url origin)

# Build all binaries into dist/Common/ (strip debug info for smaller binaries)
build: winres build-server build-client

# Generate Windows resource files (icon + manifest) from winres/icon.ico
winres:
	cd cmd/dev-null-server && go-winres simply --icon winres/icon.ico
	cd cmd/dev-null-client && go-winres simply --icon winres/icon.ico

build-server:
	go build -ldflags="-s -w -X 'main.buildCommit=$(GIT_COMMIT)' -X 'main.buildDate=$(BUILD_DATE)' -X 'main.buildRemote=$(GIT_REMOTE)'" -o dist/Common/DevNullServer.exe ./cmd/dev-null-server
	go build -ldflags="-s -w" -o dist/Common/PinggyHelper.exe ./cmd/pinggy-helper

build-client:
	go build -ldflags="-s -w -X 'main.buildCommit=$(GIT_COMMIT)' -X 'main.buildDate=$(BUILD_DATE)' -X 'main.buildRemote=$(GIT_REMOTE)'" -o dist/Common/DevNullClient.exe ./cmd/dev-null-client
	git rev-parse HEAD > dist/Common/.version

# Server: normal mode (SSH server + console TUI)
run-server: build-server
	powershell -ExecutionPolicy Bypass -File dist/DevNullServer.ps1 --no-update

# Server: LAN-only mode (no UPnP, no public IP, no Pinggy)
run-server-lan: build-server
	powershell -ExecutionPolicy Bypass -File dist/DevNullServer.ps1 --no-update --lan

# Client: connect to a running server
run-client: build-client
	powershell -ExecutionPolicy Bypass -File dist/DevNull.ps1 --no-update

# Client: local mode (headless SSH server + graphical client)
run-client-local: build-client
	powershell -ExecutionPolicy Bypass -File dist/DevNull.ps1 --no-update --local

# Run all tests
test:
	go test -v ./...

# Testbed: minimal wish+bubbletea repro binary (no product code, SSH artifact isolation)
build-testbed:
	go build -o dist/testbed.exe ./testbed

# Testbed: SSH mode without ONLCR fix (expect staircase on non-Windows)
run-testbed: build-testbed
	./dist/testbed.exe

# Testbed: SSH mode with ONLCR fix applied
run-testbed-onlcr: build-testbed
	./dist/testbed.exe --onlcr

# Generate bundle manifest for dist/Common/ assets
generate-manifest:
	go run ./cmd/gen-manifest dist/Common/ > dist/Common/.bundle-manifest.json

# Remove build outputs from dist/ (keeps Games/, Fonts/, logs/)
clean:
	rm -f dist/Common/DevNullServer.exe dist/Common/DevNullClient.exe dist/Common/PinggyHelper.exe dist/testbed.exe
