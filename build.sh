#!/usr/bin/env bash
set -euo pipefail

# ─── Корень проекта ────────────────────────────────────────────────────────
ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

# ─── Определение ОС ────────────────────────────────────────────────────────
IS_MAC=false
if [[ "$(uname -s)" == "Darwin" ]]; then IS_MAC=true; fi

# ─── Цвета ─────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

ok()   { printf "  ${GREEN}✓${NC} %s\n"  "$*"; }
info() { printf "  ${CYAN}→${NC} %s\n"   "$*"; }
warn() { printf "  ${YELLOW}⚠${NC} %s\n" "$*"; }
err()  { printf "  ${RED}❌${NC} %s\n"   "$*"; }

# ─── sed совместимый с macOS и Linux ───────────────────────────────────────
portable_sed() {
    if $IS_MAC; then
        sed -i '' "$@"
    else
        sed -i "$@"
    fi
}

# ─── Чтение конфига ────────────────────────────────────────────────────────
read_config() {
    python3 -c "import json; print(json.load(open('config/app.json'))$1)" 2>/dev/null || echo "$2"
}
VERSION=$(read_config "['version']" "1.2")
ICON_BG=$(read_config "['icon']['background_color']" "#6E1319")
ICON_FG=$(read_config "['icon']['foreground_color']" "#EDB953")

printf "${YELLOW}"
printf "  ╔═════════════════════════════════════╗\n"
printf "  ║  Lionheart v%-5s — сборка          ║\n" "$VERSION"
printf "  ╚═════════════════════════════════════╝\n"
printf "${NC}\n"

MODE="${1:-all}"
OUT="$ROOT/output"
mkdir -p "$OUT"

# ═══════════════════════════════════════════════════════════════════════════
# Вспомогательные функции
# ═══════════════════════════════════════════════════════════════════════════

human_name() {
    local os="$1" arch="$2"
    case "$os" in
        linux)   os_name="linux"   ;;
        darwin)  os_name="macos"   ;;
        windows) os_name="windows" ;;
        freebsd) os_name="freebsd" ;;
        *)       os_name="$os"     ;;
    esac
    case "$arch" in
        amd64) arch_name="x64"   ;;
        arm64) arch_name="arm64" ;;
        386)   arch_name="x86"   ;;
        *)     arch_name="$arch" ;;
    esac
    echo "lionheart-${VERSION}-${os_name}-${arch_name}"
}

check_go() {
    command -v go >/dev/null 2>&1 || { err "Go не найден. https://go.dev/dl/"; exit 1; }
    printf "  Go:  %s\n" "$(go version | awk '{print $3}')"
}

# ─── Поиск Java 17 или 21 (Kotlin 2.x не поддерживает Java 22+) ───────────
# ─── Поиск Java 17 или 21 (Kotlin 2.x не поддерживает Java 22+) ───────────
setup_java() {
    if command -v java >/dev/null 2>&1; then
        local ver=""
        # Добавляем || true, чтобы pipefail не убил скрипт, если grep ничего не найдет
        ver=$(java -version 2>&1 | grep -oE '[0-9]+\.[0-9]+' | head -1 | cut -d. -f1 || true)

        if [[ -n "$ver" ]]; then
            # Java 8 имеет формат "1.8" → после cut получаем "1"
            if [[ "$ver" == "1" ]]; then ver=8; fi
            if [[ "$ver" -ge 17 && "$ver" -le 21 ]] 2>/dev/null; then
                ok "Java $ver (совместима)"
                return 0
            fi
            warn "Java $ver несовместима с Kotlin 2.x (нужна 17–21). Ищу альтернативу..."
        fi
    fi

    # Ищем Java 17/21 в стандартных местах
    local candidates=(
        "/opt/homebrew/opt/openjdk@21/libexec/openjdk.jdk/Contents/Home"
        "/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home"
        "/Library/Java/JavaVirtualMachines/temurin-21.jdk/Contents/Home"
        "/Library/Java/JavaVirtualMachines/temurin-17.jdk/Contents/Home"
        "/Library/Java/JavaVirtualMachines/zulu-21.jdk/Contents/Home"
        "/usr/lib/jvm/java-21-openjdk-arm64"
        "/usr/lib/jvm/java-21-openjdk-amd64"
        "/usr/lib/jvm/java-17-openjdk-amd64"
    )
    for p in "${candidates[@]}"; do
        if [[ -d "$p" ]]; then
            export JAVA_HOME="$p"
            export PATH="$JAVA_HOME/bin:$PATH"
            ok "Java: $JAVA_HOME"
            return 0
        fi
    done

    # Устанавливаем через Homebrew (только macOS)
    if $IS_MAC && command -v brew >/dev/null 2>&1; then
        warn "Устанавливаю openjdk@21 через Homebrew..."
        brew install openjdk@21 --quiet 2>/dev/null || true
        local brew_java=""
        # Защита от падения, если brew --prefix завершится с ошибкой
        brew_java="$(brew --prefix openjdk@21 2>/dev/null || true)/libexec/openjdk.jdk/Contents/Home"
        if [[ -d "$brew_java" ]]; then
            export JAVA_HOME="$brew_java"
            export PATH="$JAVA_HOME/bin:$PATH"
            ok "Java 21 установлена"
            return 0
        fi
    fi

    warn "Java 17–21 не найдена. Сборка может упасть. Рекомендуется: brew install openjdk@21"
}

# ─── Поиск Android SDK ─────────────────────────────────────────────────────
find_android_sdk() {
    if [[ -z "${ANDROID_HOME:-}" || ! -d "${ANDROID_HOME:-}" ]]; then
        for p in \
            "$HOME/.android-sdk" \
            "$HOME/Library/Android/sdk" \
            "$HOME/Android/Sdk" \
            "$HOME/Android" \
            "/opt/android-sdk" \
            "/usr/local/lib/android/sdk"
        do
            if [[ -d "$p" ]]; then
                ANDROID_HOME="$p"
                break
            fi
        done
    fi

    if [[ -z "${ANDROID_HOME:-}" || ! -d "${ANDROID_HOME:-}" ]]; then
        err "Android SDK не найден."
        printf "   Укажи путь: export ANDROID_HOME=/путь/к/sdk\n"
        return 1
    fi

    export ANDROID_HOME
    export PATH="$PATH:$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools:$(go env GOPATH)/bin"
    printf "  SDK: %s\n" "$ANDROID_HOME"
}

# ─── Поиск Android NDK ─────────────────────────────────────────────────────
find_ndk() {
    if [[ -z "${ANDROID_NDK_HOME:-}" || ! -d "${ANDROID_NDK_HOME:-}" ]]; then
        if [[ -d "$ROOT/android-ndk-r25c" ]]; then
            ANDROID_NDK_HOME="$ROOT/android-ndk-r25c"
        elif [[ -d "${ANDROID_HOME:-}/ndk" ]]; then
            ANDROID_NDK_HOME=$(ls -d "$ANDROID_HOME/ndk"/*/ 2>/dev/null | sort -V | tail -1 | sed 's/\/$//')
        elif [[ -d "${ANDROID_HOME:-}/ndk-bundle" ]]; then
            ANDROID_NDK_HOME="$ANDROID_HOME/ndk-bundle"
        fi
    fi

    export ANDROID_NDK_HOME

    if [[ -z "${ANDROID_NDK_HOME:-}" || ! -d "${ANDROID_NDK_HOME:-}" ]]; then
        warn "NDK не найден!"
        printf "   export ANDROID_NDK_HOME=/путь/к/ndk\n"
        printf "   ИЛИ распакуй android-ndk-r25c рядом с проектом\n"
        return 1
    fi

    printf "  NDK: %s\n" "$ANDROID_NDK_HOME"
    return 0
}

# ─── Синхронизация конфига → Android ───────────────────────────────────────
sync_config() {
    info "Синхронизация config → Android..."
    local ANDROID="$ROOT/mobile/android"
    local RES="$ANDROID/app/src/main/res"

    if [[ -f "$ROOT/config/translations/en/strings.xml" ]]; then
        mkdir -p "$RES/values"
        cp "$ROOT/config/translations/en/strings.xml" "$RES/values/strings.xml"
    fi
    for LANG_DIR in "$ROOT/config/translations"/*/; do
        LANG=$(basename "$LANG_DIR")
        if [[ "$LANG" == "en" ]]; then continue; fi
        if [[ -f "$LANG_DIR/strings.xml" ]]; then
            mkdir -p "$RES/values-$LANG"
            cp "$LANG_DIR/strings.xml" "$RES/values-$LANG/strings.xml"
        fi
    done

    mkdir -p "$RES/values"
    cat > "$RES/values/colors.xml" << COLEOF
<?xml version="1.0" encoding="utf-8"?>
<resources>
    <color name="ic_launcher_background">${ICON_BG}</color>
</resources>
COLEOF

    for ICON_FILE in \
        "$RES/drawable/ic_launcher_foreground.xml" \
        "$RES/drawable/ic_launcher_monochrome.xml"
    do
        if [[ -f "$ICON_FILE" ]]; then
            portable_sed "s/android:fillColor=\"#[0-9A-Fa-f]\{6\}\"/android:fillColor=\"${ICON_FG}\"/g" "$ICON_FILE"
        fi
    done

    if [[ -d "$ROOT/core" ]]; then
        rm -rf "$ANDROID/core"
        cp -r "$ROOT/core" "$ANDROID/core"
    fi

    local GF="$ANDROID/app/build.gradle.kts"
    if [[ -f "$GF" ]] && ! grep -q "signingConfigs" "$GF"; then
        portable_sed '/buildTypes {/i\
    signingConfigs {\
        create("release") {\
            storeFile = signingConfigs.getByName("debug").storeFile\
            storePassword = "android"\
            keyAlias = "androiddebugkey"\
            keyPassword = "android"\
        }\
    }' "$GF"
        portable_sed '/isMinifyEnabled = true/a\
            signingConfig = signingConfigs.getByName("release")' "$GF"
    fi

    echo "sdk.dir=$ANDROID_HOME" > "$ANDROID/local.properties"

    # gradle.properties — фикс Kotlin daemon crash на Java 22+
    cat > "$ANDROID/gradle.properties" << 'GPEOF'
# AndroidX
android.useAndroidX=true
android.enableJetifier=true
android.defaults.buildfeatures.buildconfig=true

# Gradle JVM heap + открываем модули для Java 17+
org.gradle.jvmargs=-Xmx3g -Xms512m -XX:+HeapDumpOnOutOfMemoryError --add-opens=java.base/java.lang=ALL-UNNAMED --add-opens=java.base/java.util=ALL-UNNAMED --add-opens=java.base/java.io=ALL-UNNAMED

# Kotlin: отключаем инкрементальный daemon (фикс краша на Java 22+)
kotlin.incremental=false
kotlin.daemon.jvm.options=-Xmx2g --add-opens=java.base/java.lang=ALL-UNNAMED --add-opens=java.base/java.util=ALL-UNNAMED

# Параллельная сборка
org.gradle.parallel=true
org.gradle.caching=true
org.gradle.configureondemand=true
GPEOF

    ok "Конфиг синхронизирован"
}

# ─── Генерация gradlew если отсутствует ────────────────────────────────────
ensure_gradlew() {
    local ANDROID="$ROOT/mobile/android"

    if [[ -f "$ANDROID/gradlew" ]]; then
        chmod +x "$ANDROID/gradlew"
        ok "gradlew найден"
        return 0
    fi

    warn "gradlew не найден — генерирую через 'gradle wrapper'..."

    if ! command -v gradle >/dev/null 2>&1; then
        err "gradle не установлен."
        $IS_MAC && printf "   Установи: brew install gradle\n" || printf "   Установи: sudo apt install gradle\n"
        exit 1
    fi

    # Читаем нужную версию из wrapper.properties если файл есть
    local WRAPPER_PROPS="$ANDROID/gradle/wrapper/gradle-wrapper.properties"
    local GRADLE_VER="8.9"
    if [[ -f "$WRAPPER_PROPS" ]]; then
        local extracted
        extracted=$(grep "distributionUrl" "$WRAPPER_PROPS" | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1)
        if [[ -n "$extracted" ]]; then GRADLE_VER="$extracted"; fi
    fi

    cd "$ANDROID"
    gradle wrapper --gradle-version="$GRADLE_VER" --quiet
    chmod +x gradlew
    cd "$ROOT"

    ok "gradlew создан (Gradle $GRADLE_VER)"
}

# ─── Сборка Go-библиотеки (.aar) ───────────────────────────────────────────
build_golib() {
    local ANDROID="$ROOT/mobile/android"
    local AAR="$ANDROID/app/libs/liblionheart.aar"

    info "gomobile bind..."

    cd "$ANDROID/golib"
    go mod tidy 2>/dev/null || true

    local MOBILE_VER
    MOBILE_VER=$(grep "golang.org/x/mobile" go.mod 2>/dev/null | head -1 | awk '{print $2}')
    if [[ -z "$MOBILE_VER" ]]; then MOBILE_VER="latest"; fi

    go install "golang.org/x/mobile/cmd/gomobile@$MOBILE_VER" 2>/dev/null \
        || go install golang.org/x/mobile/cmd/gomobile@latest
    go install "golang.org/x/mobile/cmd/gobind@$MOBILE_VER" 2>/dev/null \
        || go install golang.org/x/mobile/cmd/gobind@latest

    gomobile init 2>/dev/null || true
    go get golang.org/x/mobile/bind 2>/dev/null || true
    go mod tidy 2>/dev/null || true

    mkdir -p "$ANDROID/app/libs"
    gomobile bind \
        -target=android \
        -androidapi=24 \
        -ldflags="-s -w" \
        -o "$AAR" \
        .

    ok ".aar собран"
    cd "$ROOT"   # ВАЖНО: всегда возвращаемся в корень
}

# ─── Запуск Gradle ─────────────────────────────────────────────────────────
run_gradle() {
    local TASK="$1"
    local ANDROID="$ROOT/mobile/android"

    cd "$ANDROID"
    info "$TASK..."
    ./gradlew "$TASK" --no-daemon

    local EXIT=$?
    cd "$ROOT"   # ВАЖНО: всегда возвращаемся в корень
    return $EXIT
}

# ─── Копирование APK в output/ ─────────────────────────────────────────────
copy_apk() {
    local TYPE="$1"
    local ANDROID="$ROOT/mobile/android"
    local SUFFIX=""
    if [[ "$TYPE" == "debug" ]]; then SUFFIX="-debug"; fi

    local APK
    APK=$(find "$ANDROID" -name "*.apk" -path "*/${TYPE}/*" 2>/dev/null | head -1)

    if [[ -n "$APK" ]]; then
        local DEST="$OUT/lionheart-${VERSION}-android${SUFFIX}.apk"
        cp "$APK" "$DEST"
        ok "$(basename "$DEST") ($(du -sh "$DEST" | cut -f1))"

        if command -v adb >/dev/null 2>&1 && adb devices 2>/dev/null | grep -q "device$"; then
            info "adb install ($TYPE)..."
            adb install -r "$DEST" 2>/dev/null && ok "Установлено на устройство" || true
        fi
    else
        warn "APK не найден для типа: $TYPE"
    fi
}

# ═══════════════════════════════════════════════════════════════════════════
# Режимы сборки
# ═══════════════════════════════════════════════════════════════════════════

build_cli() {
    printf "\n${BOLD}[CLI] Сборка...${NC}\n"
    check_go

    cd "$ROOT/cmd/lionheart"
    go mod tidy 2>/dev/null || true

    local PLATFORMS=(
        "linux/amd64"   "linux/arm64"
        "darwin/amd64"  "darwin/arm64"
        "windows/amd64" "windows/arm64"
        "freebsd/amd64"
    )

    for P in "${PLATFORMS[@]}"; do
        local OS="${P%/*}" ARCH="${P#*/}" SUFFIX=""
        if [[ "$OS" == "windows" ]]; then SUFFIX=".exe"; fi
        local NAME
        NAME="$(human_name "$OS" "$ARCH")${SUFFIX}"
        info "$NAME"
        CGO_ENABLED=0 GOOS="$OS" GOARCH="$ARCH" \
            go build -ldflags="-s -w" -o "$OUT/$NAME" . 2>/dev/null \
            || warn "пропущено: $NAME"
    done

    cd "$ROOT"
    ok "CLI → output/"
}

build_debugapk() {
    printf "\n${BOLD}[APK] Android debug...${NC}\n"
    check_go
    setup_java
    find_android_sdk || return 1
    find_ndk || { err "Невозможно собрать .aar без NDK."; return 1; }
    sync_config
    ensure_gradlew

    local ANDROID="$ROOT/mobile/android"
    local AAR="$ANDROID/app/libs/liblionheart.aar"
    local NEED_AAR=0

    if [[ ! -f "$AAR" ]]; then NEED_AAR=1; fi
    if [[ "$NEED_AAR" == "0" ]]; then
        local NEWEST_GO
        NEWEST_GO=$(find "$ANDROID/golib" -name "*.go" -newer "$AAR" 2>/dev/null | head -1)
        if [[ -n "$NEWEST_GO" ]]; then NEED_AAR=1; fi
    fi

    if [[ "$NEED_AAR" == "1" ]]; then
        build_golib
    else
        ok ".aar актуален, пропускаю gomobile bind"
    fi

    run_gradle assembleDebug
    copy_apk debug
}

build_apk() {
    printf "\n${BOLD}[APK] Android (debug + release)...${NC}\n"
    check_go
    setup_java
    find_android_sdk || return 1
    find_ndk || { err "Невозможно собрать .aar без NDK."; return 1; }
    sync_config
    ensure_gradlew
    build_golib
    run_gradle assembleDebug
    copy_apk debug
    run_gradle assembleRelease
    copy_apk release
}

case "$MODE" in
    cli)      build_cli ;;
    debugapk) build_debugapk ;;
    apk)      build_apk ;;
    all)      build_cli; build_apk ;;
    *)
        printf "Использование: ./build.sh [cli|apk|debugapk|all]\n\n"
        printf "  cli       — только CLI бинарники (все платформы)\n"
        printf "  debugapk  — только debug APK + adb install (быстро)\n"
        printf "  apk       — debug + release APK\n"
        printf "  all       — всё (CLI + APK)\n"
        exit 1
        ;;
esac

printf "\n${GREEN}═══════════════════════════════════════${NC}\n"
printf "${GREEN} ✅ Готово! Всё в output/:${NC}\n"
printf "${GREEN}═══════════════════════════════════════${NC}\n"
ls -lh "$OUT"/ 2>/dev/null
printf "\n"