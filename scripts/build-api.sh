#!/bin/bash

# SMAP Notification API - Build and Push to Zot Registry
# Usage: ./scripts/build-api.sh [build-push|login|help]

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# ── Configuration ────────────────────────────────────────────────────────────
REGISTRY="${HARBOR_REGISTRY:-registry.tantai.dev}"
HARBOR_USER="${HARBOR_USERNAME:?HARBOR_USERNAME is not set}"
HARBOR_PASS="${HARBOR_PASSWORD:?HARBOR_PASSWORD is not set. Export it in ~/.zshrc}"
PROJECT="smap"
SERVICE="notification-api"
DOCKERFILE="cmd/api/Dockerfile"
PLATFORM="${PLATFORM:-linux/amd64}"

# ── Helpers ──────────────────────────────────────────────────────────────────
info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[OK]${NC} $1"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $1"; }
err()     { echo -e "${RED}[ERROR]${NC} $1"; }

generate_tag() { date +"%y%m%d-%H%M%S"; }

image_name() {
    local tag="${1:-$(generate_tag)}"
    echo "${REGISTRY}/${PROJECT}/${SERVICE}:${tag}"
}

# ── Login ────────────────────────────────────────────────────────────────────
login() {
    info "Logging into Zot registry: $REGISTRY"
    echo "$HARBOR_PASS" | docker login "$REGISTRY" -u "$HARBOR_USER" --password-stdin
    success "Logged in to $REGISTRY"
}

# ── Prerequisites ────────────────────────────────────────────────────────────
check_prereqs() {
    command -v docker &>/dev/null || { err "Docker not installed"; exit 1; }
    docker buildx version &>/dev/null || { err "Docker buildx not available"; exit 1; }
    [ -f "$DOCKERFILE" ] || { err "Dockerfile not found: $DOCKERFILE"; exit 1; }
}

# ── Build & Push ─────────────────────────────────────────────────────────────
build_and_push() {
    check_prereqs
    login

    local tag
    tag=$(generate_tag)
    local img
    img=$(image_name "$tag")
    local latest
    latest=$(image_name "latest")

    info "Registry:   $REGISTRY"
    info "Image:      $img"
    info "Platform:   $PLATFORM"
    info "Dockerfile: $DOCKERFILE"
    echo ""

    docker buildx build \
        --platform "$PLATFORM" \
        --provenance=false \
        --sbom=false \
        --tag "$img" \
        --tag "$latest" \
        --file "$DOCKERFILE" \
        --push \
        .

    echo ""
    success "Pushed: $img"
    success "Pushed: $latest"
}

# ── Help ─────────────────────────────────────────────────────────────────────
show_help() {
    cat <<EOF
${GREEN}SMAP Notification API - Build & Push (Zot Registry)${NC}

Usage: $0 [command]

Commands:
    build-push   Build and push image (default)
    login        Login to Zot registry
    help         Show this help

Environment Variables:
    ZOT_REGISTRY   Registry URL     (default: 172.16.21.10:5000)
    ZOT_USERNAME   Registry user    (required)
    ZOT_PASSWORD   Registry pass    (required)
    PLATFORM       Target platform  (default: linux/amd64)

Image Format:
    ${REGISTRY}/${PROJECT}/${SERVICE}:<YYMMDD-HHMMSS>
    ${REGISTRY}/${PROJECT}/${SERVICE}:latest
EOF
}

# ── Main ─────────────────────────────────────────────────────────────────────
case "${1:-build-push}" in
    build-push) build_and_push ;;
    login)      login ;;
    help|--help|-h) show_help ;;
    *)
        err "Unknown command: $1"
        show_help
        exit 1
        ;;
esac
