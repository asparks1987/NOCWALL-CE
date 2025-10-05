#!/usr/bin/env bash
#
# Multi-architecture Docker build helper.
# Builds the UISP NOC container for multiple CPU targets and pushes it to Docker Hub.
#
# Usage:
#   ./build-multiarch.sh --image youruser/uisp-noc:tag [options]
#
# Options:
#   --image <name>        Docker image (including tag) to build and push. (required)
#   --context <path>      Build context directory. Default: current directory.
#   --platforms <list>    Comma-separated platforms list. Default: linux/amd64,linux/arm64,linux/arm/v7
#   --builder <name>      buildx builder name to use/create. Default: uisp-noc-multiarch
#   --push                Push image to registry (default behaviour).
#   --load                Load image into local Docker instead of pushing (mutually exclusive with --push).
#   --pull                Always attempt to pull newer base images.
#   --no-cache            Disable Docker build cache.
#   --build-arg KEY=VAL   Additional build argument(s); may be specified multiple times.
#   -h, --help            Show this help message.
#
# Example:
#   ./build-multiarch.sh --image predheadtx/uisp-noc:latest --build-arg BUILD_SHA=$(git rev-parse HEAD)

set -euo pipefail

show_help() {
  sed -n '2,32p' "$0"
}

image=""
context="."
platforms="linux/amd64,linux/arm64,linux/arm/v7"
builder="uisp-noc-multiarch"
push=true
load=false
pull=false
no_cache=false
build_args=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --image)
      image="$2"
      shift 2
      ;;
    --context)
      context="$2"
      shift 2
      ;;
    --platforms)
      platforms="$2"
      shift 2
      ;;
    --builder)
      builder="$2"
      shift 2
      ;;
    --push)
      push=true
      load=false
      shift
      ;;
    --load)
      load=true
      push=false
      shift
      ;;
    --pull)
      pull=true
      shift
      ;;
    --no-cache)
      no_cache=true
      shift
      ;;
    --build-arg)
      build_args+=("--build-arg" "$2")
      shift 2
      ;;
    -h|--help)
      show_help
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      show_help
      exit 1
      ;;
  esac
done

if [[ -z "$image" ]]; then
  echo "Error: --image <name> is required." >&2
  show_help
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "Error: docker CLI not found in PATH." >&2
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  echo "Error: unable to communicate with Docker daemon." >&2
  exit 1
fi

echo "Using builder: $builder"
if ! docker buildx inspect "$builder" >/dev/null 2>&1; then
  echo "Creating buildx builder '$builder'"
  docker buildx create --name "$builder" --use >/dev/null
else
  docker buildx use "$builder" >/dev/null
fi

echo "Bootstrapping builder (if required)..."
docker buildx inspect "$builder" --bootstrap >/dev/null

cmd=(docker buildx build --platform "$platforms" -t "$image")

if $push; then
  cmd+=(--push)
elif $load; then
  cmd+=(--load)
else
  cmd+=(--push)
fi

if $pull; then
  cmd+=(--pull)
fi

if $no_cache; then
  cmd+=(--no-cache)
fi

if [[ ${#build_args[@]} -gt 0 ]]; then
  cmd+=("${build_args[@]}")
fi

cmd+=("$context")

echo
echo "Running: ${cmd[*]}"
echo

"${cmd[@]}"

echo
echo "Multi-arch build completed successfully." >&2

