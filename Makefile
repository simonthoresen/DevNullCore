.PHONY: build build-server build-client build-test-game run-server run-server-lan run-server-local run-client run-client-local run-test-game test clean

ifeq ($(OS),Windows_NT)
  GIT_COMMIT  := $(shell git rev-parse --short HEAD 2>nul || echo dev)
  BUILD_DATE  := $(shell git log -1 --format=%cI 2>nul || echo unknown)
  GIT_REMOTE  := $(shell git remote get-url origin 2>nul || echo "")
else
  GIT_COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
  BUILD_DATE  := $(shell git log -1 --format=%cI 2>/dev/null || echo unknown)
  GIT_REMOTE  := $(shell git remote get-url origin 2>/dev/null || echo "")
endif

# Build all binaries into dist/ (strip debug info for smaller binaries)
build: build-server build-client

build-server:
	go build -ldflags="-s -w -X 'main.buildCommit=$(GIT_COMMIT)' -X 'main.buildDate=$(BUILD_DATE)' -X 'main.buildRemote=$(GIT_REMOTE)'" -o dist/dev-null-server.exe ./cmd/dev-null-server
	go build -ldflags="-s -w" -o dist/pinggy-helper.exe ./cmd/pinggy-helper

build-client:
	go build -ldflags="-s -w -X 'main.buildCommit=$(GIT_COMMIT)' -X 'main.buildDate=$(BUILD_DATE)' -X 'main.buildRemote=$(GIT_REMOTE)'" -o dist/dev-null-client.exe ./cmd/dev-null-client

build-test-game:
	go build -o dist/test-game.exe ./cmd/test-game

# Server: normal mode (SSH server + console TUI)
run-server: build-server
	./dist/dev-null-server.exe --data-dir dist

# Server: LAN-only mode (no UPnP, no public IP, no Pinggy)
run-server-lan: build-server
	./dist/dev-null-server.exe --data-dir dist --lan

# Server: local mode (headless SSH server + terminal client)
run-server-local: build-server
	./dist/dev-null-server.exe --data-dir dist --local

# Client: connect to a running server
run-client: build-client
	./dist/dev-null-client.exe

# Client: local mode (headless SSH server + graphical client)
run-client-local: build-client
	./dist/dev-null-client.exe --data-dir dist --local

# Test: direct chrome rendering with no SSH (GAME=cube|crawler|etc.)
run-test-game: build-test-game
	./dist/test-game.exe --data-dir dist --game $(GAME)

# Run all tests
test:
	go test -v ./...

# Remove build outputs from dist/ (keeps games/, fonts/, logs/)
clean:
	rm -f dist/dev-null-server.exe dist/dev-null-client.exe dist/pinggy-helper.exe dist/test-game.exe
