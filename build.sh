#!/bin/bash
# Build script for GOVM cross-platform binary release
# Usage: ./build.sh [version]

set -e

VERSION=${1:-"dev"}
BUILD_DIR="build"
RELEASE_DIR="release"

# Platforms to build for
PLATFORMS=(
  "linux:amd64"
  "linux:arm64"
  "windows:amd64"
  "windows:arm64"
  "darwin:amd64"
  "darwin:arm64"
)

# Output binary names
declare -A BINARY_NAMES=(
  ["linux-amd64"]="govm-linux-amd64"
  ["linux-arm64"]="govm-linux-arm64"
  ["windows-amd64"]="govm-windows-amd64.exe"
  ["windows-arm64"]="govm-windows-arm64.exe"
  ["darwin-amd64"]="govm-macos-amd64"
  ["darwin-arm64"]="govm-macos-arm64"
)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Building GOVM v${VERSION}${NC}"
echo "=========================================="

# Clean and create directories
rm -rf "${BUILD_DIR}" "${RELEASE_DIR}"
mkdir -p "${BUILD_DIR}" "${RELEASE_DIR}"

# Build for each platform
for platform in "${PLATFORMS[@]}"; do
  IFS=':' read -r os arch <<< "$platform"

  binary_key="${os}-${arch}"
  binary_name="${BINARY_NAMES[$binary_key]}"

  echo -e "${YELLOW}Building ${os}/${arch}...${NC}"

  # Build binary
  export GOOS="${os}"
  export GOARCH="${arch}"
  export CGO_ENABLED=0

  if go build -o "${BUILD_DIR}/${binary_name}" \
    -ldflags="-X main.Version=${VERSION}" \
    ./cmd; then

    echo -e "${GREEN}✓ Built ${binary_name}${NC}"

    # Calculate SHA256
    cd "${BUILD_DIR}"
    sha256sum "${binary_name}" > "${binary_name}.sha256"
    echo -e "${GREEN}✓ Created ${binary_name}.sha256${NC}"
    cd - > /dev/null

    # Copy to release directory
    cp "${BUILD_DIR}/${binary_name}" "${RELEASE_DIR}/"
    cp "${BUILD_DIR}/${binary_name}.sha256" "${RELEASE_DIR}/"

  else
    echo -e "${RED}✗ Failed to build ${binary_name}${NC}"
    exit 1
  fi
done

# Create checksums file
echo -e "${YELLOW}Creating release checksums...${NC}"
cd "${RELEASE_DIR}"
cat ./*.sha256 > SHA256SUMS
echo -e "${GREEN}✓ Created SHA256SUMS${NC}"
cd - > /dev/null

# Summary
echo ""
echo -e "${GREEN}=========================================="
echo "Build Complete!"
echo "==========================================${NC}"
echo ""
echo "Release binaries are in: ${RELEASE_DIR}/"
echo ""
ls -lh "${RELEASE_DIR}"
echo ""
echo "Built binaries:"
for binary_key in "${!BINARY_NAMES[@]}"; do
  binary_name="${BINARY_NAMES[$binary_key]}"
  if [ -f "${RELEASE_DIR}/${binary_name}" ]; then
    size=$(ls -lh "${RELEASE_DIR}/${binary_name}" | awk '{print $5}')
    echo -e "${GREEN}✓${NC} ${binary_name} (${size})"
  fi
done
echo ""
echo "To verify integrity:"
echo "  cd ${RELEASE_DIR}"
echo "  sha256sum -c SHA256SUMS"
