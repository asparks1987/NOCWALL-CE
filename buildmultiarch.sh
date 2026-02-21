#!/usr/bin/env bash
set -euo pipefail

# buildmultiarch build runner.
# Actions:
#   /install   - prepare build host (docker/buildx/builder, optional docker login)
#   /preflight - run release checks (ce gate + syntax + compose config)
#   /build     - multi-arch build + push web and API images
#   /update    - git pull (if repo) then /build
#
# Defaults:
#   Docker Hub user: predheadtx
#   Web repo:        nocwall
#   API repo:        nocwall-api
#   Tag:             latest

DOCKERHUB_USER="${DOCKERHUB_USER:-predheadtx}"
WEB_REPO="${WEB_REPO:-nocwall}"
API_REPO="${API_REPO:-nocwall-api}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64,linux/arm/v7}"
BUILDER="${BUILDER:-nocwall-dockerhub}"
WEB_DOCKERFILE="${WEB_DOCKERFILE:-Dockerfile}"
API_DOCKERFILE="${API_DOCKERFILE:-api/Dockerfile}"
DRY_RUN=0
SKIP_LOGIN=0
SKIP_PREFLIGHT=0
SKIP_GIT_SHA_TAG=0

usage() {
  cat <<'EOF'
Usage:
  ./buildmultiarch.sh /install [options]
  ./buildmultiarch.sh /preflight [options]
  ./buildmultiarch.sh /build   [options]
  ./buildmultiarch.sh /update  [options]
  ./buildmultiarch.sh [options]

Options:
  --u, -u <dockerhub-user>   Docker Hub namespace (default: predheadtx)
  --user <dockerhub-user>    Docker Hub namespace (default: predheadtx)
  --i, -i <repo:tag>         Shortcut for web image repo+tag (example: nocwall:latest)
  --web-repo <repo>          Web image repository (default: nocwall)
  --api-repo <repo>          API image repository (default: nocwall-api)
  --name <repo>              Back-compat alias for --web-repo
  --tag <tag>                Image tag (default: latest)
  --platforms <list>         Build platforms (default: linux/amd64,linux/arm64,linux/arm/v7)
  --builder <name>           buildx builder name (default: nocwall-dockerhub)
  --web-dockerfile <path>    Web Dockerfile path (default: Dockerfile)
  --api-dockerfile <path>    API Dockerfile path (default: api/Dockerfile)
  --dry-run                  Print commands without executing build/push
  --skip-login               Skip docker login during /install
  --skip-preflight           Skip preflight checks before /build
  --no-git-sha-tag           Disable extra image tag: git-<shortsha>
  -h, --help                 Show help

Examples:
  ./buildmultiarch.sh /install
  ./buildmultiarch.sh /preflight
  ./buildmultiarch.sh /build
  ./buildmultiarch.sh --u predheadtx -i nocwall:latest
  ./buildmultiarch.sh /build --user predheadtx --web-repo nocwall --api-repo nocwall-api --tag latest
EOF
}

log() {
  printf '[buildmultiarch] %s\n' "$*"
}

warn() {
  printf '[buildmultiarch][WARN] %s\n' "$*" >&2
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
      --u|-u)
        DOCKERHUB_USER="$2"; shift 2 ;;
      --user)
        DOCKERHUB_USER="$2"; shift 2 ;;
      --i|-i)
        parse_image_shortcut "$2"; shift 2 ;;
      --web-repo)
        WEB_REPO="$2"; shift 2 ;;
      --api-repo)
        API_REPO="$2"; shift 2 ;;
      --name)
        WEB_REPO="$2"; shift 2 ;;
      --tag)
        IMAGE_TAG="$2"; shift 2 ;;
      --platforms)
        PLATFORMS="$2"; shift 2 ;;
      --builder)
        BUILDER="$2"; shift 2 ;;
      --web-dockerfile)
        WEB_DOCKERFILE="$2"; shift 2 ;;
      --api-dockerfile)
        API_DOCKERFILE="$2"; shift 2 ;;
      --dry-run)
        DRY_RUN=1; shift ;;
      --skip-login)
        SKIP_LOGIN=1; shift ;;
      --skip-preflight)
        SKIP_PREFLIGHT=1; shift ;;
      --no-git-sha-tag)
        SKIP_GIT_SHA_TAG=1; shift ;;
      -h|--help)
        usage; exit 0 ;;
      *)
        die "Unknown option: $1"
        ;;
    esac
  done
}

parse_image_shortcut() {
  local spec="$1"
  local repo tag
  repo="${spec%%:*}"
  tag="${spec#*:}"
  if [[ -z "$repo" || -z "$tag" || "$repo" == "$spec" ]]; then
    die "Invalid image spec for -i/--i: '$spec' (expected repo:tag)"
  fi
  WEB_REPO="$repo"
  IMAGE_TAG="$tag"
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
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]'
}

build_image_ref() {
  local repo="$1"
  printf '%s/%s:%s' "$DOCKERHUB_USER" "$(repo_slug "$repo")" "$IMAGE_TAG"
}

current_git_short_sha() {
  if command -v git >/dev/null 2>&1 && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    git rev-parse --short HEAD 2>/dev/null || true
  fi
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

preflight_action() {
  local failed=0
  log "Running preflight checks."

  if [[ -f scripts/ce-release-gate.sh ]]; then
    if [[ "$DRY_RUN" == "1" ]]; then
      printf '[DRY-RUN] bash scripts/ce-release-gate.sh\n'
    elif ! bash scripts/ce-release-gate.sh; then
      failed=1
    fi
  else
    warn "scripts/ce-release-gate.sh not found; skipping."
  fi

  if [[ -f buildmultiarch.sh ]]; then
    if [[ "$DRY_RUN" == "1" ]]; then
      printf '[DRY-RUN] bash -n buildmultiarch.sh\n'
    elif ! bash -n buildmultiarch.sh; then
      failed=1
    fi
  fi
  if [[ -f NOCWALL.sh ]]; then
    if [[ "$DRY_RUN" == "1" ]]; then
      printf '[DRY-RUN] bash -n NOCWALL.sh\n'
    elif ! bash -n NOCWALL.sh; then
      failed=1
    fi
  fi

  if command -v node >/dev/null 2>&1 && [[ -f assets/app.js ]]; then
    if [[ "$DRY_RUN" == "1" ]]; then
      printf '[DRY-RUN] node --check assets/app.js\n'
    elif ! node --check assets/app.js; then
      failed=1
    fi
  else
    warn "Node or assets/app.js missing; skipping JS syntax check."
  fi

  if command -v docker >/dev/null 2>&1 && [[ -f docker-compose.yml ]]; then
    if [[ "$DRY_RUN" == "1" ]]; then
      printf '[DRY-RUN] docker compose config >/dev/null\n'
    elif ! docker compose config >/dev/null; then
      failed=1
    fi
  else
    warn "Docker CLI or docker-compose.yml missing; skipping compose validation."
  fi

  if command -v go >/dev/null 2>&1 && [[ -f api/go.mod ]]; then
    if [[ "$DRY_RUN" == "1" ]]; then
      printf '[DRY-RUN] (cd api && go test ./...)\n'
    elif ! (cd api && go test ./...); then
      failed=1
    fi
  else
    warn "Go toolchain or api/go.mod missing; skipping API tests."
  fi

  if [[ "$failed" -ne 0 ]]; then
    die "Preflight checks failed."
  fi
  log "Preflight checks passed."
}

build_single_image() {
  local repo="$1"
  local dockerfile="$2"
  local image
  local sha_tag=""
  local sha=""
  local tag_args=()
  image="$(build_image_ref "$repo")"
  [[ -f "$dockerfile" ]] || die "Dockerfile not found: $dockerfile"

  tag_args=(-t "$image")
  if [[ "$SKIP_GIT_SHA_TAG" != "1" ]]; then
    sha="$(current_git_short_sha)"
    if [[ -n "$sha" ]]; then
      sha_tag="${DOCKERHUB_USER}/$(repo_slug "$repo"):git-${sha}"
      tag_args+=(-t "$sha_tag")
    fi
  fi

  log "Publishing image: $image (dockerfile: $dockerfile)"
  run_cmd docker buildx build \
    --platform "$PLATFORMS" \
    -f "$dockerfile" \
    "${tag_args[@]}" \
    --push \
    .
  log "Build/push complete: $image"
  if [[ -n "$sha_tag" ]]; then
    log "Build/push complete: $sha_tag"
  fi
}

build_images_action() {
  build_single_image "$WEB_REPO" "$WEB_DOCKERFILE"
  build_single_image "$API_REPO" "$API_DOCKERFILE"

  log "Published images:"
  log "  $(build_image_ref "$WEB_REPO")"
  log "  $(build_image_ref "$API_REPO")"
}

build_action() {
  ensure_docker
  ensure_builder
  if [[ "$SKIP_PREFLIGHT" != "1" ]]; then
    preflight_action
  fi
  build_images_action
}

update_action() {
  ensure_docker
  if command -v git >/dev/null 2>&1 && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    log "Pulling latest git changes."
    run_cmd git pull --ff-only
  else
    warn "Not in a git repo (or git missing); skipping git pull."
  fi
  build_action
}

main() {
  local action="${1:-/build}"
  if [[ "$action" == -* ]]; then
    parse_common_args "$@"
    action="/build"
  else
    shift || true
    parse_common_args "$@"
  fi

  case "$action" in
    /install|install)
      install_action
      ;;
    /preflight|preflight)
      preflight_action
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
