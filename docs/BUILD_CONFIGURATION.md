# Build Configuration Guide

| [🚀 Overview](../README.md) | [⚙️ Environment](ENVIRONMENT_VARIABLES.md) | [🛠️ Build](BUILD_CONFIGURATION.md) | [🌐 Networking](NETWORKING.md) | [🤖 Architecture](ARCHITECTURE_VISION.md) |
| :---: | :---: | :---: | :---: | :---: |

This guide explains the build requirements, process, and troubleshooting for M3TAL.

## 📋 Build Requirements

### Go Environment

M3TAL requires **Go 1.21 or higher** to build.

#### Check Your Go Version

```bash
go version
```

**Expected output:**

```text
go version go1.21.x linux/amd64
```

#### Install Go (Linux/macOS)

```bash
# Download and install Go
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz

# Add to PATH (add this to your ~/.bashrc or ~/.zshrc)
export PATH=$PATH:/usr/local/go/bin

# Verify installation
go version
```

#### Install Go (macOS with Homebrew)

```bash
brew install go
```

### Go Modules Dependencies

M3TAL uses Go modules. All dependencies are listed in `go.mod`.

**Download dependencies:**

```bash
go mod download
go mod tidy
```

## 🛠️ Build Process

### 1. Clone the Repository

```bash
git clone https://github.com/jakej985-rgb/M3tal-Media-Server.git
cd M3tal-Media-Server
```

### 2. Build the Binary

#### Option A: Using build.sh (Linux/macOS)

```bash
chmod +x build.sh
./build.sh
```

#### Option B: Using Makefile

```bash
make build
```

#### Option C: Manual Build

```bash
go build -o m3tal ./cmd/m3tal/main.go
```

A successful build produces:

```text
Build completed successfully
Binary location: ./m3tal
Binary size: ~15 MB
```

## 📊 Build Output

A successful build looks like:

```bash
$ ./build.sh
📥 Downloading dependencies...
🚀 Building M3TAL CLI...
     → Writes: ./m3tal (the main CLI binary)
🚀 Building M3TAL API...
     → Writes: ./m3tal-api (the backend API binary)
✅ Build complete. Binaries: ./m3tal, ./m3tal-api
```

The build produces two binaries:

| Binary | Purpose | Size (approx) |
| :--- | :--- | :--- |
| `./m3tal` | Main CLI orchestrator | ~14-16 MB |
| `./m3tal-api` | Backend API server | ~12-14 MB |

## 🔍 Build Logs

If the build fails, check:

### Build Log Location

```bash
# Check for build.log
ls -la build.log

# Or check stderr
./build.sh 2> build-error.log
```

### Common Build Errors

#### Error: "go: command not found"

**Cause**: Go is not installed or not in PATH.

**Solution**:

```bash
# Verify Go installation
which go

# Or check PATH
echo $PATH | grep go
```

#### Error: "module not found"

**Cause**: Go modules not downloaded.

**Solution**:

```bash
go mod download
go mod tidy
./build.sh
```

#### Error: "permission denied"

**Cause**: File permissions issue.

**Solution**:

```bash
chmod +x build.sh
./build.sh
```

## 🎯 Build Script Details

### What build.sh Does

1. **Checks Go installation** - Verifies `go version`
2. **Verifies dependencies** - Runs `go mod verify`
3. **Builds binary** - Compiles with `go build`
4. **Optimizes** - Strips debug info for smaller binary

### Build Flags

```bash
# The build script uses these flags:
go build \
    -ldflags "-s -w" \  # Strip debug info
    -o m3tal \            # Output file
    ./cmd/m3tal/main.go   # Entry point
```

## 📦 Cross-Platform Builds

### Build for Different Platforms

**Linux (amd64)**:

```bash
GOOS=linux GOARCH=amd64 go build -o m3tal-linux ./cmd/m3tal/main.go
```

**macOS (amd64)**:

```bash
GOOS=darwin GOARCH=amd64 go build -o m3tal-macos ./cmd/m3tal/main.go
```

**macOS (arm64)**:

```bash
GOOS=darwin GOARCH=arm64 go build -o m3tal-macos-arm ./cmd/m3tal/main.go
```

**Windows**:

```bash
GOOS=windows GOARCH=amd64 go build -o m3tal.exe ./cmd/m3tal/main.go
```

### Make Cross-Compile Easy

Add to your `~/.bashrc`:

```bash
build-m3tal() {
    local platform=$1
    local ext=""
    [ "$platform" = "windows" ] && ext=".exe"
    GOOS=$platform GOARCH=amd64 go build -o "m3tal-${platform}${ext}" ./cmd/m3tal/main.go
}
```

## 🔧 Docker Build (Alternative)

If you don't want to install Go locally:

```bash
# Build in Docker container
docker run --rm -v $(pwd):/app golang:1.21-alpine \
    sh -c "cd /app && go mod download && go build -o m3tal ./cmd/m3tal/main.go"

# Or use the provided Dockerfile
docker build -t m3tal-builder .
docker run --rm -v $(pwd):/output m3tal-builder cp /app/m3tal /output/m3tal
```

## ✅ Pre-Build Checklist

Before building, verify:

- [ ] Go 1.21+ installed: `go version`
- [ ] Dependencies installed: `go mod download`
- [ ] Sufficient disk space: `df -h .`
- [ ] Write permissions: `touch test && rm test`

## 🚀 After Build

After successful build:

1. **Verify binary works**:

   ```bash
   ./m3tal version
   ```

2. **Check file size** (should be ~15MB):

   ```bash
   ls -lh m3tal
   ```

3. **Test basic commands**:

   ```bash
   ./m3tal help
   ```

## 📝 Notes

- The binary is **statically linked** - no external dependencies needed
- Build time: ~10-30 seconds (depends on CPU)
- Binary size: ~14-16 MB after stripping
- Debug builds: Remove `-ldflags "-s -w"` for debugging info
