#!/bin/bash
# lionheart build script — компиляция под все платформы
#
# Требования: Go 1.22+
# Установка: https://go.dev/dl/
#
# Использование:
#   chmod +x build.sh && ./build.sh

set -e

MODULE="lionheart"
OUT="dist"
mkdir -p "$OUT"

echo "lionheart — сборка"
echo ""

# Скачиваем зависимости
echo "→ go mod tidy..."
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

for PLATFORM in "${PLATFORMS[@]}"; do
    GOOS="${PLATFORM%/*}"
    GOARCH="${PLATFORM#*/}"
    SUFFIX=""
    if [ "$GOOS" = "windows" ]; then
        SUFFIX=".exe"
    fi
    OUTPUT="${OUT}/lionheart-${GOOS}-${GOARCH}${SUFFIX}"
    echo "→ ${GOOS}/${GOARCH}..."
    CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build -ldflags="-s -w" -o "$OUTPUT" .
done

echo ""
echo "✓ Готово! Бинарники в ${OUT}/:"
ls -lh "$OUT"/
echo ""
echo "Для текущей платформы:  go build -o lionheart ."