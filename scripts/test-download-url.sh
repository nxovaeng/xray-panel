#!/bin/bash

# Test script to verify download URLs

set -e

GITHUB_REPO="${GITHUB_REPO:-nxovaeng/xray-panel}"
VERSION="${1:-v1.0.0}"
ARCH="amd64"

echo "Testing download URLs..."
echo "Repository: $GITHUB_REPO"
echo "Version: $VERSION"
echo "Architecture: $ARCH"
echo ""

# Test 1: Specific version URL
echo "Test 1: Specific version URL"
URL="https://github.com/$GITHUB_REPO/releases/download/${VERSION}/xray-panel-${VERSION}-linux-${ARCH}.tar.gz"
echo "URL: $URL"
if curl -I -L "$URL" 2>&1 | grep -q "200 OK\|302 Found"; then
    echo "✓ URL is accessible"
else
    echo "✗ URL is not accessible"
fi
echo ""

# Test 2: Get latest version from API
echo "Test 2: Get latest version from API"
LATEST_VERSION=$(curl -s "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | grep -oP '"tag_name": "\K(.*)(?=")')
if [[ -n "$LATEST_VERSION" ]]; then
    echo "✓ Latest version: $LATEST_VERSION"
    
    # Test latest version URL
    LATEST_URL="https://github.com/$GITHUB_REPO/releases/download/${LATEST_VERSION}/xray-panel-${LATEST_VERSION}-linux-${ARCH}.tar.gz"
    echo "URL: $LATEST_URL"
    if curl -I -L "$LATEST_URL" 2>&1 | grep -q "200 OK\|302 Found"; then
        echo "✓ Latest version URL is accessible"
    else
        echo "✗ Latest version URL is not accessible"
    fi
else
    echo "✗ Failed to get latest version"
fi
echo ""

# Test 3: List all releases
echo "Test 3: List available releases"
RELEASES=$(curl -s "https://api.github.com/repos/$GITHUB_REPO/releases" | grep -oP '"tag_name": "\K(.*)(?=")' | head -5)
if [[ -n "$RELEASES" ]]; then
    echo "Available releases:"
    echo "$RELEASES"
else
    echo "No releases found or API error"
fi
echo ""

# Test 4: Check release assets
echo "Test 4: Check release assets for $VERSION"
ASSETS=$(curl -s "https://api.github.com/repos/$GITHUB_REPO/releases/tags/${VERSION}" | grep -oP '"name": "\K(.*)(?=")' | grep "\.tar\.gz\|\.zip")
if [[ -n "$ASSETS" ]]; then
    echo "Available assets:"
    echo "$ASSETS"
else
    echo "No assets found for version $VERSION"
fi
echo ""

# Test 5: Verify URL format
echo "Test 5: Verify URL format"
echo "Expected format:"
echo "  https://github.com/USER/REPO/releases/download/VERSION/xray-panel-VERSION-OS-ARCH.tar.gz"
echo ""
echo "Example URLs:"
echo "  https://github.com/$GITHUB_REPO/releases/download/v1.0.0/xray-panel-v1.0.0-linux-amd64.tar.gz"
echo "  https://github.com/$GITHUB_REPO/releases/download/v1.0.0/xray-panel-v1.0.0-linux-arm64.tar.gz"
echo "  https://github.com/$GITHUB_REPO/releases/download/v1.0.0/xray-panel-v1.0.0-windows-amd64.zip"
echo "  https://github.com/$GITHUB_REPO/releases/download/v1.0.0/xray-panel-v1.0.0-darwin-amd64.tar.gz"
echo ""

echo "Test completed!"
