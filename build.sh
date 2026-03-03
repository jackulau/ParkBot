#!/usr/bin/env bash
#
# build.sh — Cross-platform build script for ParkBot
#
# Builds ParkBot binaries for macOS, Linux, and Windows.
#
# Fyne uses CGO for platform-native graphics (OpenGL, Cocoa, Win32, X11),
# so cross-compilation requires appropriate C cross-compilers. When a
# required cross-compiler is not found, that target is skipped with a
# warning rather than failing the entire build.
#
# Usage:
#   ./build.sh                  # Build all available targets
#   ./build.sh darwin            # Build macOS targets only
#   ./build.sh linux             # Build Linux targets only
#   ./build.sh windows           # Build Windows targets only
#   ./build.sh clean             # Remove build output directory
#
# Environment variables:
#   OUTDIR       — Output directory (default: ./dist)
#   LDFLAGS      — Extra ldflags (default: "-s -w" for stripped binaries)
#   VERSION      — Version string baked into the binary name (default: "dev")
#   CC_LINUX_AMD64   — C compiler for linux/amd64 (default: auto-detect)
#   CC_LINUX_ARM64   — C compiler for linux/arm64 (default: auto-detect)
#   CC_WINDOWS_AMD64 — C compiler for windows/amd64 (default: auto-detect)
#

set -euo pipefail

# ─── Configuration ───────────────────────────────────────────────────────────

OUTDIR="${OUTDIR:-./dist}"
LDFLAGS="${LDFLAGS:--s -w}"
VERSION="${VERSION:-dev}"
APP_NAME="parkbot"

# Colors for output (disabled if not a terminal)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
else
    RED='' GREEN='' YELLOW='' BLUE='' NC=''
fi

# ─── Helpers ──────────────────────────────────────────────────────────────────

info()    { printf "${BLUE}[INFO]${NC}  %s\n" "$*"; }
success() { printf "${GREEN}[OK]${NC}    %s\n" "$*"; }
warn()    { printf "${YELLOW}[SKIP]${NC}  %s\n" "$*"; }
fail()    { printf "${RED}[FAIL]${NC}  %s\n" "$*"; }

# Find a working C cross-compiler from a list of candidates
find_cc() {
    for cc in "$@"; do
        if command -v "$cc" >/dev/null 2>&1; then
            echo "$cc"
            return 0
        fi
    done
    return 1
}

# Build a single target
build_target() {
    local goos="$1" goarch="$2" cc="${3:-}" suffix=""
    local outname="${APP_NAME}-${goos}-${goarch}"

    if [ "$goos" = "windows" ]; then
        suffix=".exe"
        outname="${outname}${suffix}"
    fi

    local outpath="${OUTDIR}/${outname}"

    info "Building ${goos}/${goarch} ..."

    local env_vars=(
        "CGO_ENABLED=1"
        "GOOS=${goos}"
        "GOARCH=${goarch}"
    )

    if [ -n "$cc" ]; then
        env_vars+=("CC=${cc}")
    fi

    if env "${env_vars[@]}" go build \
        -ldflags "${LDFLAGS}" \
        -o "${outpath}" \
        . 2>&1; then

        local size
        size=$(du -h "${outpath}" | cut -f1 | xargs)
        success "${outname} (${size})"
        return 0
    else
        fail "${goos}/${goarch} build failed"
        return 1
    fi
}

# ─── Clean ────────────────────────────────────────────────────────────────────

if [ "${1:-}" = "clean" ]; then
    info "Removing ${OUTDIR}/"
    rm -rf "${OUTDIR}"
    success "Clean complete"
    exit 0
fi

# ─── Setup ────────────────────────────────────────────────────────────────────

mkdir -p "${OUTDIR}"

FILTER="${1:-all}"  # "all", "darwin", "linux", or "windows"

info "ParkBot build — version=${VERSION}"
info "Output directory: ${OUTDIR}"
info "Go version: $(go version | awk '{print $3}')"
echo ""

BUILT=0
SKIPPED=0
FAILED=0

# ─── macOS builds ─────────────────────────────────────────────────────────────

if [ "$FILTER" = "all" ] || [ "$FILTER" = "darwin" ]; then
    # macOS builds work natively on macOS (Apple's toolchain handles both archs)
    if [ "$(uname -s)" = "Darwin" ]; then
        if build_target darwin amd64; then
            BUILT=$((BUILT + 1))
        else
            FAILED=$((FAILED + 1))
        fi

        if build_target darwin arm64; then
            BUILT=$((BUILT + 1))
        else
            FAILED=$((FAILED + 1))
        fi
    else
        warn "darwin/amd64 — skipped (requires macOS host)"
        warn "darwin/arm64 — skipped (requires macOS host)"
        SKIPPED=$((SKIPPED + 2))
    fi
fi

# ─── Linux builds ────────────────────────────────────────────────────────────

if [ "$FILTER" = "all" ] || [ "$FILTER" = "linux" ]; then
    # linux/amd64
    if [ "$(uname -s)" = "Linux" ] && [ "$(uname -m)" = "x86_64" ]; then
        # Native build on Linux x86_64
        if build_target linux amd64; then
            BUILT=$((BUILT + 1))
        else
            FAILED=$((FAILED + 1))
        fi
    else
        CC_LINUX="${CC_LINUX_AMD64:-}"
        if [ -z "$CC_LINUX" ]; then
            CC_LINUX=$(find_cc \
                x86_64-linux-musl-gcc \
                x86_64-linux-gnu-gcc \
                x86_64-unknown-linux-gnu-gcc \
            ) || true
        fi
        if [ -n "$CC_LINUX" ]; then
            if build_target linux amd64 "$CC_LINUX"; then
                BUILT=$((BUILT + 1))
            else
                FAILED=$((FAILED + 1))
            fi
        else
            warn "linux/amd64 — skipped (no C cross-compiler found; install musl-cross or see README)"
            SKIPPED=$((SKIPPED + 1))
        fi
    fi

    # linux/arm64
    if [ "$(uname -s)" = "Linux" ] && [ "$(uname -m)" = "aarch64" ]; then
        # Native build on Linux arm64
        if build_target linux arm64; then
            BUILT=$((BUILT + 1))
        else
            FAILED=$((FAILED + 1))
        fi
    else
        CC_LINUX_A="${CC_LINUX_ARM64:-}"
        if [ -z "$CC_LINUX_A" ]; then
            CC_LINUX_A=$(find_cc \
                aarch64-linux-musl-gcc \
                aarch64-linux-gnu-gcc \
                aarch64-unknown-linux-gnu-gcc \
            ) || true
        fi
        if [ -n "$CC_LINUX_A" ]; then
            if build_target linux arm64 "$CC_LINUX_A"; then
                BUILT=$((BUILT + 1))
            else
                FAILED=$((FAILED + 1))
            fi
        else
            warn "linux/arm64 — skipped (no C cross-compiler found; install musl-cross or see README)"
            SKIPPED=$((SKIPPED + 1))
        fi
    fi
fi

# ─── Windows builds ──────────────────────────────────────────────────────────

if [ "$FILTER" = "all" ] || [ "$FILTER" = "windows" ]; then
    # windows/amd64
    CC_WIN="${CC_WINDOWS_AMD64:-}"
    if [ -z "$CC_WIN" ]; then
        CC_WIN=$(find_cc \
            x86_64-w64-mingw32-gcc \
            i686-w64-mingw32-gcc \
        ) || true
    fi
    if [ -n "$CC_WIN" ]; then
        if build_target windows amd64 "$CC_WIN"; then
            BUILT=$((BUILT + 1))
        else
            FAILED=$((FAILED + 1))
        fi
    else
        warn "windows/amd64 — skipped (no MinGW cross-compiler found; install mingw-w64 or see README)"
        SKIPPED=$((SKIPPED + 1))
    fi
fi

# ─── Summary ──────────────────────────────────────────────────────────────────

echo ""
echo "─────────────────────────────────────────"
info "Build summary:"
info "  Built:   ${BUILT}"
info "  Skipped: ${SKIPPED}"
info "  Failed:  ${FAILED}"

if [ "$BUILT" -gt 0 ]; then
    echo ""
    info "Output files:"
    ls -lh "${OUTDIR}/" | tail -n +2
fi

echo ""
if [ "$FAILED" -gt 0 ]; then
    fail "Some builds failed"
    exit 1
elif [ "$BUILT" -eq 0 ]; then
    warn "No targets were built (missing cross-compilers?)"
    exit 1
else
    success "All available targets built successfully"
    exit 0
fi
