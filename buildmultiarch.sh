#!/usr/bin/env bash
set -euo pipefail

# buildmultiarch master command runner.
# Actions:
#   /install  - prepare build host (docker/buildx/builder, optional docker login)
#   /build    - multi-arch build + push to Docker Hub
#   /update   - git pull (if repo) then /build
#
# Defaults:
#   Docker Hub user: predheadtx
#   Image name:      NOCWALL (published as lowercase repository "nocwall")
#   Tag:             latest

DOCKERHUB_USER="${DOCKERHUB_USER:-predheadtx}"
IMAGE_NAME="${IMAGE_NAME:-NOCWALL}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64,linux/arm/v7}"
BUILDER="${BUILDER:-nocwall-dockerhub}"
DRY_RUN=0
SKIP_LOGIN=0

usage() {
  cat <<'EOF'
Usage:
  ./buildmultiarch.sh /install [options]
  ./buildmultiarch.sh /build   [options]
  ./buildmultiarch.sh /update  [options]

Options:
  --user <dockerhub-user>    Docker Hub namespace (default: predheadtx)
  --name <image-name>        Image name (default: NOCWALL)
  --tag <tag>                Tag (default: latest)
  --platforms <list>         Build platforms (default: linux/amd64,linux/arm64,linux/arm/v7)
  --builder <name>           buildx builder name (default: nocwall-dockerhub)
  --dry-run                  Print commands without executing build/push
  --skip-login               Skip docker login during /install
  -h, --help                 Show help

Examples:
  ./buildmultiarch.sh /install
  ./buildmultiarch.sh /build
  ./buildmultiarch.sh /update
  ./buildmultiarch.sh /build --user predheadtx --name NOCWALL --tag latest
EOF
}

log() {
  printf '[buildmultiarch] %s\n' "$*"
}

die() {
  printf '[buildmultiarch][ERROR] %s\n' "$*" >&2
  exit 1
}

run_cmd() {
  if [[ "$DRY_RUN" == "1" ]]; then
    printf '[DRY-RUN] %s\n' "$*"
    return 0
  fi
  "$@"
}

parse_common_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --user)
        DOCKERHUB_USER="$2"; shift 2 ;;
      --name)
        IMAGE_NAME="$2"; shift 2 ;;
      --tag)
        IMAGE_TAG="$2"; shift 2 ;;
      --platforms)
        PLATFORMS="$2"; shift 2 ;;
      --builder)
        BUILDER="$2"; shift 2 ;;
      --dry-run)
        DRY_RUN=1; shift ;;
      --skip-login)
        SKIP_LOGIN=1; shift ;;
      -h|--help)
        usage; exit 0 ;;
      *)
        die "Unknown option: $1"
        ;;
    esac
  done
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "Required command not found: $1"
}

ensure_docker() {
  require_cmd docker
  docker info >/dev/null 2>&1 || die "Docker daemon not reachable."
}

ensure_builder() {
  local exists=0
  if docker buildx inspect "$BUILDER" >/dev/null 2>&1; then
    exists=1
  fi

  if [[ "$exists" == "1" ]]; then
    run_cmd docker buildx use "$BUILDER" >/dev/null
  else
    log "Creating buildx builder: $BUILDER"
    run_cmd docker buildx create --name "$BUILDER" --driver docker-container --use >/dev/null
  fi

  run_cmd docker buildx inspect "$BUILDER" --bootstrap >/dev/null
}

repo_slug() {
  printf '%s' "$IMAGE_NAME" | tr '[:upper:]' '[:lower:]'
}

build_image_ref() {
  printf '%s/%s:%s' "$DOCKERHUB_USER" "$(repo_slug)" "$IMAGE_TAG"
}

install_action() {
  ensure_docker
  ensure_builder

  # QEMU/binfmt for cross-platform builds (best-effort).
  log "Ensuring binfmt emulation support."
  run_cmd docker run --privileged --rm tonistiigi/binfmt --install all >/dev/null || true

  if [[ "$SKIP_LOGIN" == "1" ]]; then
    log "Skipping docker login as requested."
  else
    log "Running docker login (Docker Hub)."
    run_cmd docker login
  fi

  log "Install step complete."
}

build_action() {
  ensure_docker
  ensure_builder

  local image
  image="$(build_image_ref)"
  log "Publishing CE image to Docker Hub: $image"
  run_cmd docker buildx build \
    --platform "$PLATFORMS" \
    -f Dockerfile \
    -t "$image" \
    --push \
    .
  log "Build/push complete: $image"
}

update_action() {
  ensure_docker
  if command -v git >/dev/null 2>&1 && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    log "Pulling latest git changes."
    run_cmd git pull --ff-only
  else
    log "Not in a git repo (or git missing); skipping git pull."
  fi
  build_action
}

main() {
  local action="${1:-}"
  if [[ -z "$action" ]]; then
    usage
    exit 1
  fi
  shift || true
  parse_common_args "$@"

  case "$action" in
    /install|install)
      install_action
      ;;
    /build|build)
      build_action
      ;;
    /update|update)
      update_action
      ;;
    /help|help|-h|--help)
      usage
      ;;
    *)
      die "Unknown action: $action"
      ;;
  esac
}

main "$@"
