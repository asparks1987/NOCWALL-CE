#!/usr/bin/env bash
# Multi-arch builder for UISP NOC app and API. Builds and pushes both images.
# Usage:
#   ./build-multiarch.sh --author youruser --tag beta [options]
# Options:
#   --author <name>        (required) registry namespace/author
#   --tag <tag>            (required) image tag
#   --platforms <list>     platforms (default: linux/amd64,linux/arm64)
#   --builder <name>       buildx builder (default: uisp-noc-multiarch)
#   --push                 push images (default)
#   --load                 load into local Docker (mutually exclusive with --push)
#   --pull                 pull latest bases
#   --no-cache             disable cache
#   --build-arg KEY=VAL    repeatable build args (applied to both builds)
#   -h|--help              show this help
set -euo pipefail

show_help() { sed -n '2,120p' "$0"; }

author=""
tag=""
platforms="linux/amd64,linux/arm64"
builder="uisp-noc-multiarch"
push=true
load=false
pull=false
no_cache=false
build_args=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --author) author="$2"; shift 2 ;;
    --tag) tag="$2"; shift 2 ;;
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

if [[ -z "$author" || -z "$tag" ]]; then
  echo "Error: --author and --tag are required." >&2
  show_help
  exit 1
fi

app_image="${author}/uisp-noc:${tag}"
api_image="${author}/uisp-api:${tag}"

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

common_opts=(--platform "$platforms")
if $push; then common_opts+=(--push); fi
if $load; then common_opts+=(--load); fi
if $pull; then common_opts+=(--pull); fi
if $no_cache; then common_opts+=(--no-cache); fi
if [[ ${#build_args[@]} -gt 0 ]]; then common_opts+=("${build_args[@]}"); fi

echo
echo "Building app image: $app_image"
app_cmd=(docker buildx build "${common_opts[@]}" -t "$app_image" -f Dockerfile .)
echo "Running: ${app_cmd[*]}"
"${app_cmd[@]}"

echo
echo "Building API image: $api_image"
api_cmd=(docker buildx build "${common_opts[@]}" -t "$api_image" -f api/Dockerfile .)
echo "Running: ${api_cmd[*]}"
"${api_cmd[@]}"

echo
echo "Builds complete." >&2
