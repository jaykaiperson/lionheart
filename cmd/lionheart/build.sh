#!/bin/bash
# Build lionheart CLI for all platforms
set -e
cd "$(dirname "$0")"

echo "lionheart CLI — build"
echo ""

go mod tidy

PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
    "android/arm64"
    "freebsd/amd64"
)

OUT="../../dist"
mkdir -p "$OUT"

for PLATFORM in "${PLATFORMS[@]}"; do
    GOOS="${PLATFORM%/*}"
    GOARCH="${PLATFORM#*/}"
    SUFFIX=""
    if [ "$GOOS" = "windows" ]; then SUFFIX=".exe"; fi
    OUTPUT="$OUT/lionheart-${GOOS}-${GOARCH}${SUFFIX}"
    echo "→ ${GOOS}/${GOARCH}..."
    CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build -ldflags="-s -w" -o "$OUTPUT" .
done

echo ""
echo "✓ Done! Binaries in dist/:"
ls -lh "$OUT"/
