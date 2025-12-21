#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# GitHub repository
REPO="MironCo/indiekku"
BINARY_NAME="indiekku"
INSTALL_DIR="/usr/local/bin"

# Print fancy header
clear
echo -e "${BLUE}${BOLD}"
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║                                                           ║"
echo "║                    🎮  INDIEKKU  🎮                       ║"
echo "║                                                           ║"
echo "║            Unity Server Orchestration Tool                ║"
echo "║                                                           ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo -e "${NC}"
echo ""

# Check if Docker is installed
echo -e "${BOLD}[1/4] Checking dependencies...${NC}"
if ! command -v docker &> /dev/null; then
    echo -e "${YELLOW}Docker not found. Installing Docker...${NC}"
    curl -fsSL https://get.docker.com -o get-docker.sh
    sh get-docker.sh
    rm get-docker.sh
    systemctl start docker
    systemctl enable docker
    echo -e "${GREEN}✓ Docker installed${NC}"
else
    echo -e "${GREEN}✓ Docker found${NC}"
fi

# Check if Docker is running
if ! docker info &> /dev/null 2>&1; then
    echo -e "${YELLOW}Starting Docker...${NC}"
    systemctl start docker 2>/dev/null || true
    sleep 2
    if ! docker info &> /dev/null 2>&1; then
        echo -e "${RED}✗ Docker is not running. Please start Docker and try again.${NC}"
        exit 1
    fi
fi
echo -e "${GREEN}✓ Docker is running${NC}"
echo ""

# Detect OS and architecture
echo -e "${BOLD}[2/4] Detecting platform...${NC}"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)
        echo -e "${RED}✗ Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

echo -e "${GREEN}✓ Platform: $OS-$ARCH${NC}"
echo ""

# Download latest release
echo -e "${BOLD}[3/4] Downloading indiekku...${NC}"
LATEST_VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    echo -e "${RED}✗ Could not fetch latest version${NC}"
    exit 1
fi

echo -e "Latest version: ${BLUE}$LATEST_VERSION${NC}"

DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/${BINARY_NAME}-${OS}-${ARCH}.tar.gz"
TMP_DIR=$(mktemp -d)

curl -L -o "$TMP_DIR/${BINARY_NAME}.tar.gz" "$DOWNLOAD_URL" 2>&1 | grep -v "%" || true

if [ $? -ne 0 ]; then
    echo -e "${RED}✗ Failed to download${NC}"
    rm -rf "$TMP_DIR"
    exit 1
fi

echo -e "${GREEN}✓ Downloaded${NC}"
echo ""

# Install
echo -e "${BOLD}[4/4] Installing...${NC}"
tar -xzf "$TMP_DIR/${BINARY_NAME}.tar.gz" -C "$TMP_DIR"
TMP_FILE="$TMP_DIR/$BINARY_NAME"

if [ ! -f "$TMP_FILE" ]; then
    echo -e "${RED}✗ Binary not found in archive${NC}"
    rm -rf "$TMP_DIR"
    exit 1
fi

chmod +x "$TMP_FILE"

if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
else
    sudo mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
fi

rm -rf "$TMP_DIR"

if ! command -v $BINARY_NAME &> /dev/null; then
    echo -e "${RED}✗ Installation failed${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Installed to $INSTALL_DIR/$BINARY_NAME${NC}"
echo ""

# Success message
echo -e "${GREEN}${BOLD}"
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║                                                           ║"
echo "║              ✨ Installation Complete! ✨                 ║"
echo "║                                                           ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo -e "${NC}"
echo ""
echo -e "${BOLD}Quick Start:${NC}"
echo -e "  ${BLUE}1.${NC} indiekku serve              ${GREEN}# Start the server${NC}"
echo -e "  ${BLUE}2.${NC} open http://localhost:8080  ${GREEN}# Open web UI${NC}"
echo -e "  ${BLUE}3.${NC} Upload your Unity build     ${GREEN}# Deploy!${NC}"
echo ""
echo -e "Documentation: ${BLUE}https://github.com/$REPO${NC}"
echo ""
