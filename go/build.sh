#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

RELEASE_DIR="release"
rm -rf "$RELEASE_DIR"
mkdir -p "$RELEASE_DIR"

echo "Building m for all platforms..."

VERSION=$(git describe --tags --always 2>/dev/null || echo "dev")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS="-s -w -X main.version=$VERSION -X main.commit=$COMMIT"

PLATFORMS=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
)

for PLATFORM in "${PLATFORMS[@]}"; do
    OS=$(echo "$PLATFORM" | cut -d'/' -f1)
    ARCH=$(echo "$PLATFORM" | cut -d'/' -f2)

    PLATFORM_DIR="$RELEASE_DIR/$OS-$ARCH"
    mkdir -p "$PLATFORM_DIR"

    OUTPUT_NAME="m"
    if [ "$OS" = "windows" ]; then
        OUTPUT_NAME="m.exe"
    fi

    echo "  Building: $OS/$ARCH..."
    GOOS=$OS GOARCH=$ARCH go build -ldflags="$LDFLAGS" -o "../$PLATFORM_DIR/$OUTPUT_NAME" ./cmd/task
    echo "    -> $PLATFORM_DIR/$OUTPUT_NAME"
done

echo ""
echo "Building Android platforms..."

ANDROID_PLATFORMS=(
    "android/arm64"
    "android/arm"
)

for ARCH in "${ANDROID_PLATFORMS[@]}"; do
    PLATFORM_DIR="$RELEASE_DIR/$ARCH"
    mkdir -p "$PLATFORM_DIR"

    ANDROID_OS=$(echo "$ARCH" | cut -d'/' -f1)
    ANDROID_ARCH=$(echo "$ARCH" | cut -d'/' -f2)

    echo "  Building: $ARCH..."
    if [ "$ANDROID_ARCH" = "arm" ]; then
        CC="zig cc -target arm-linux-gnueabihf" CXX="zig c++ -target arm-linux-gnueabihf" GOOS=linux GOARCH=$ANDROID_ARCH go build -ldflags="$LDFLAGS" -o "../$PLATFORM_DIR/m" ./cmd/task
    else
        CGO_ENABLED=0 GOOS=$ANDROID_OS GOARCH=$ANDROID_ARCH go build -ldflags="$LDFLAGS" -o "../$PLATFORM_DIR/m" ./cmd/task
    fi
    echo "    -> $PLATFORM_DIR/m"
done

echo ""
echo "Build complete!"
find "$RELEASE_DIR" -type f -name "m*" | xargs ls -lh
