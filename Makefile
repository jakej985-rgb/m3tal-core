# M3TAL Core Makefile

.PHONY: build clean deb help

VERSION := $(shell cat VERSION 2>/dev/null | sed 's/^v//' || echo "0.0.0")

help:
	@echo "M3TAL Core Management Utility"
	@echo ""
	@echo "Build:"
	@echo "  make build    - Build Web GUI and API daemon"
	@echo "  make deb      - Build binaries & package into .deb"
	@echo "  make clean    - Remove compiled binaries and .deb artifacts"

build:
	@echo "🎨 Building Web GUI..."
	cd gui && npm run build
	@echo "🚀 Building M3TAL Core API daemon..."
	VERSION=$(VERSION) ./scripts/build-go.sh native
	@echo "✅ Build complete."

deb:
	@./scripts/build-deb.sh

clean:
	@echo "🧹 Cleaning binaries..."
	rm -f m3tal-api
	rm -f *.deb
	@echo "✅ Cleanup complete."
