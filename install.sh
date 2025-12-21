#!/bin/bash
set -e

# indiekku installer script
# Usage: curl -sSL https://raw.githubusercontent.com/MironCo/indiekku/main/install.sh | sudo bash

echo "========================================"
echo "  indiekku installer"
echo "========================================"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: This script must be run as root (use sudo)"
    exit 1
fi

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Convert architecture names
case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "Error: Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Check if OS is supported
if [ "$OS" != "linux" ] && [ "$OS" != "darwin" ]; then
    echo "Error: Unsupported OS: $OS"
    echo "indiekku supports Linux and macOS only."
    exit 1
fi

echo "Detected platform: $OS-$ARCH"
echo ""

# Check for Docker
echo "Checking dependencies..."
if ! command -v docker &> /dev/null; then
    echo "Error: Docker is not installed."
    echo ""
    echo "Please install Docker first:"
    echo "  macOS: https://docs.docker.com/desktop/install/mac-install/"
    echo "  Linux: https://docs.docker.com/engine/install/"
    exit 1
fi

if ! docker ps &> /dev/null; then
    echo "Error: Docker is installed but not running."
    echo "Please start Docker and try again."
    exit 1
fi

echo "✓ Docker is installed and running"
echo ""

# Download latest release
REPO="MironCo/indiekku"
RELEASE_URL="https://github.com/$REPO/releases/latest/download"
FILENAME="indiekku-$OS-$ARCH.tar.gz"
DOWNLOAD_URL="$RELEASE_URL/$FILENAME"

echo "Downloading indiekku from $DOWNLOAD_URL..."
TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

if ! curl -sSL -o "$FILENAME" "$DOWNLOAD_URL"; then
    echo "Error: Failed to download indiekku"
    rm -rf "$TEMP_DIR"
    exit 1
fi

echo "✓ Downloaded successfully"
echo ""

# Extract archive
echo "Installing indiekku..."
tar -xzf "$FILENAME"

# Install binary
INSTALL_DIR="/usr/local/bin"
mkdir -p "$INSTALL_DIR"
mv indiekku "$INSTALL_DIR/indiekku"
chmod +x "$INSTALL_DIR/indiekku"

# Install web directory
WEB_DIR="/usr/local/share/indiekku"
mkdir -p "$WEB_DIR"
mv web "$WEB_DIR/"

echo "✓ Installed to $INSTALL_DIR/indiekku"
echo "✓ Web files installed to $WEB_DIR/web"
echo ""

# Cleanup
cd - > /dev/null
rm -rf "$TEMP_DIR"

# Verify installation
if ! command -v indiekku &> /dev/null; then
    echo "Warning: indiekku was installed but is not in PATH"
    echo "You may need to add $INSTALL_DIR to your PATH"
    exit 1
fi

VERSION=$(indiekku version 2>/dev/null || echo "unknown")

echo "========================================"
echo "  indiekku installed successfully!"
echo "========================================"
echo ""
echo "Version: $VERSION"
echo ""
echo "Next steps:"
echo "  1. Start the indiekku server:"
echo "     indiekku serve"
echo ""
echo "  2. Open the web UI at http://localhost:8080"
echo ""
echo "  3. Upload your Unity server build"
echo ""
echo "Documentation: https://github.com/$REPO"
echo ""
