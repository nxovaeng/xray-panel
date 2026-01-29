#!/bin/bash
# Build script for all platforms and architectures

set -e

VERSION="${1:-v1.0.0}"
OUTPUT_DIR="dist"

echo "Building Xray Panel ${VERSION}"
echo "================================"

# Clean output directory
rm -rf "${OUTPUT_DIR}"
mkdir -p "${OUTPUT_DIR}"

# Build matrix
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
    "windows/arm64"
    "darwin/amd64"
    "darwin/arm64"
)

for PLATFORM in "${PLATFORMS[@]}"; do
    GOOS="${PLATFORM%/*}"
    GOARCH="${PLATFORM#*/}"
    
    OUTPUT_NAME="panel-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}.exe"
    fi
    
    echo ""
    echo "Building ${GOOS}/${GOARCH}..."
    
    CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build \
        -v -trimpath \
        -ldflags="-s -w -X main.Version=${VERSION}" \
        -o "${OUTPUT_DIR}/${OUTPUT_NAME}" \
        ./cmd/panel
    
    if [ $? -eq 0 ]; then
        SIZE=$(du -h "${OUTPUT_DIR}/${OUTPUT_NAME}" | cut -f1)
        echo "✓ Built ${OUTPUT_NAME} (${SIZE})"
    else
        echo "✗ Failed to build ${OUTPUT_NAME}"
        exit 1
    fi
done

echo ""
echo "================================"
echo "Build Summary:"
ls -lh "${OUTPUT_DIR}/"
echo ""
echo "Total size:"
du -sh "${OUTPUT_DIR}/"
