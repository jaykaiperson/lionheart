set -e
ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'
echo -e "${YELLOW}"
echo "  ╔══════════════════════════════════════╗"
echo "  ║  Lionheart iOS v1.2 — build          ║"
echo "  ╚══════════════════════════════════════╝"
echo -e "${NC}"
command -v go >/dev/null 2>&1 || { echo -e "${RED}❌ Go не найден. brew install go${NC}"; exit 1; }
command -v gomobile >/dev/null 2>&1 || { echo -e "${RED}❌ gomobile не найден.\n   go install golang.org/x/mobile/cmd/gomobile@latest\n   gomobile init${NC}"; exit 1; }
echo -e "  Go:       $(go version | awk '{print $3}')"
echo -e "  gomobile: $(which gomobile)"
echo -e "\n${BOLD}[1/2] Golib.xcframework...${NC}"
GOLIB="$ROOT/golib-ios"
XCF="$ROOT/Golib.xcframework"
if [ -d "$XCF" ]; then
    NEWEST=$(find "$GOLIB" -name "*.go" -newer "$XCF" 2>/dev/null | head -1)
    if [ -z "$NEWEST" ]; then
        echo -e "  ${GREEN}✓ .xcframework is up to date, skipping${NC}"
    else
        echo -e "  → Rebuilding (Go changes detected)..."
        rm -rf "$XCF"
        cd "$GOLIB" && gomobile bind -target ios/arm64 -o "$XCF" . && cd "$ROOT"
        echo -e "  ${GREEN}✓ Golib.xcframework built${NC}"
    fi
else
    cd "$GOLIB" && gomobile bind -target ios/arm64 -o "$XCF" . && cd "$ROOT"
    echo -e "  ${GREEN}✓ Golib.xcframework built${NC}"
fi
echo -e "\n${BOLD}[2/2] Xcode project...${NC}"
if [ ! -d "$ROOT/Lionheart.xcodeproj" ] && [ ! -d "$ROOT/Lionheart.xcworkspace" ]; then
    echo -e "${YELLOW}  ⚠ Xcode project not found.${NC}"
    echo -e "${YELLOW}  Create a project in Xcode per README.md${NC}"
    echo -e ""
    echo -e "  Golib.xcframework is ready — now open Xcode."
    exit 0
fi
echo -e "  → xcodebuild archive..."
SCHEME="Lionheart"
xcodebuild \
    -project Lionheart.xcodeproj \
    -scheme "$SCHEME" \
    -configuration Debug \
    -sdk iphoneos \
    -archivePath "$ROOT/build/Lionheart.xcarchive" \
    archive \
    CODE_SIGN_IDENTITY="-" \
    CODE_SIGNING_REQUIRED=NO \
    CODE_SIGNING_ALLOWED=NO \
    2>&1 | tail -5
echo -e "  ${GREEN}✓ Archive${NC}"
mkdir -p "$ROOT/output"
PAYLOAD="$ROOT/build/Payload"
rm -rf "$PAYLOAD"
mkdir -p "$PAYLOAD"
cp -r "$ROOT/build/Lionheart.xcarchive/Products/Applications/Lionheart.app" "$PAYLOAD/"
cd "$ROOT/build" && zip -qr "$ROOT/output/Lionheart-1.2.ipa" Payload/ && cd "$ROOT"
rm -rf "$PAYLOAD"
echo -e "  ${GREEN}✓ output/Lionheart-1.2.ipa${NC}"
echo ""
echo -e "${GREEN}══════════════════════════════════════${NC}"
echo -e "${GREEN} ✅ Done!${NC}"
echo -e "${GREEN}══════════════════════════════════════${NC}"
echo -e "  Install: AltStore → My Apps → + → Lionheart-1.2.ipa"
echo ""
