#!/usr/bin/env bash
# Multi-arch builder for UISP NOC (backend + assets). Builds and optionally pushes.
# Usage:
#   ./build-multiarch.sh --image youruser/uisp-noc:tag [options]
# Options:
#   --image <name:tag>   (required) target image
#   --context <path>     build context (default: .)
#   --platforms <list>   platforms (default: linux/amd64,linux/arm64)
#   --builder <name>     buildx builder (default: uisp-noc-multiarch)
#   --push               push image (default)
#   --load               load into local Docker (mutually exclusive with --push)
#   --pull               pull latest bases
#   --no-cache           disable cache
#   --build-arg KEY=VAL  repeatable build args
#   -h|--help            show this help
set -euo pipefail

show_help() { sed -n '2,80p' "$0"; }

image=""
context="."
platforms="linux/amd64,linux/arm64"
builder="uisp-noc-multiarch"
push=true
load=false
pull=false
no_cache=false
build_args=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --image) image="$2"; shift 2 ;;
    --context) context="$2"; shift 2 ;;
    --platforms) platforms="$2"; shift 2 ;;
    --builder) builder="$2"; shift 2 ;;
    --push) push=true; load=false; shift ;;
    --load) load=true; push=false; shift ;;
    --pull) pull=true; shift ;;
    --no-cache) no_cache=true; shift ;;
    --build-arg) build_args+=("--build-arg" "$2"); shift 2 ;;
    -h|--help) show_help; exit 0 ;;
    *) echo "Unknown option: $1" >&2; show_help; exit 1 ;;
  esac
done

if [[ -z "$image" ]]; then
  echo "Error: --image <name:tag> is required." >&2
  show_help; exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "Error: docker CLI not found." >&2; exit 1
fi
if ! docker info >/dev/null 2>&1; then
  echo "Error: cannot talk to Docker daemon." >&2; exit 1
fi

echo "Using builder: $builder"
if ! docker buildx inspect "$builder" >/dev/null 2>&1; then
  docker buildx create --name "$builder" --use >/dev/null
else
  docker buildx use "$builder" >/dev/null
fi
docker buildx inspect "$builder" --bootstrap >/dev/null

cmd=(docker buildx build --platform "$platforms" -t "$image")

if $push; then cmd+=(--push); fi
if $load; then cmd+=(--load); fi
if $pull; then cmd+=(--pull); fi
if $no_cache; then cmd+=(--no-cache); fi
if [[ ${#build_args[@]} -gt 0 ]]; then cmd+=("${build_args[@]}"); fi

cmd+=("$context")

echo
echo "Running: ${cmd[*]}"
echo

"${cmd[@]}"

echo
echo "Build complete." >&2
