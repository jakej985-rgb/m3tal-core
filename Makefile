# M3TAL Platform Makefile

.PHONY: build clean deb up down status help

VERSION := $(shell cat VERSION 2>/dev/null | sed 's/^v//' || echo "0.0.0")

help:
	@echo "M3TAL Management Utility"
	@echo ""
	@echo "Build:"
	@echo "  make build    - Compile CLI and API binaries"
	@echo "  make deb      - Build binaries & package into .deb (requires nfpm)"
	@echo "  make clean    - Remove compiled binaries and .deb artifacts"
	@echo ""
	@echo "Runtime:"
	@echo "  make up       - Start the M3TAL stack using the Go orchestrator"
	@echo "  make down     - Stop the M3TAL stack"
	@echo "  make status   - List container status"

build:
	@echo "🎨 Building Web GUI..."
	cd gui && npm run build
	@echo "🚀 Building M3TAL CLI and API for AMD64 v1..."
	VERSION=$(VERSION) ./scripts/build-go.sh
	@echo "✅ Build complete."

deb:
	@./scripts/build-deb.sh

clean:
	@echo "🧹 Cleaning binaries..."
	rm -f m3tal m3tal-api
	rm -f *.deb
	@echo "✅ Cleanup complete."

up: build
	./m3tal up

down:
	./m3tal down

status:
	./m3tal list
