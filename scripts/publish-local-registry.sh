#!/usr/bin/env bash
set -euo pipefail

# One-command multi-arch publish for full NOCWALL-CE stack to local registry.
# Builds and pushes both images:
# - web: Dockerfile
# - api: api/Dockerfile
#
# Usage:
#   ./scripts/publish-local-registry.sh
#   REGISTRY=172.16.120.5:5000 NAMESPACE=nocwall TAG=v0.1.0 ./scripts/publish-local-registry.sh

REGISTRY="${REGISTRY:-172.16.120.5:5000}"
NAMESPACE="${NAMESPACE:-nocwall}"
TAG="${TAG:-}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64,linux/arm/v7}"
BUILDER="${BUILDER:-nocwall-multiarch}"
PULL="${PULL:-0}"
NO_CACHE="${NO_CACHE:-0}"
NO_LATEST="${NO_LATEST:-0}"
DRY_RUN="${DRY_RUN:-0}"

if ! command -v docker >/dev/null 2>&1; then
  echo "Docker CLI not found in PATH." >&2
  exit 1
fi

docker info >/dev/null 2>&1 || {
  echo "Unable to communicate with Docker daemon." >&2
  exit 1
}

if ! docker buildx inspect "$BUILDER" >/dev/null 2>&1; then
  echo "Creating buildx builder '$BUILDER'..."
  docker buildx create --name "$BUILDER" --use >/dev/null
else
  docker buildx use "$BUILDER" >/dev/null
fi
docker buildx inspect --bootstrap >/dev/null

if [[ -z "$TAG" ]]; then
  if git rev-parse --short HEAD >/dev/null 2>&1; then
    TAG="git-$(git rev-parse --short HEAD)"
  else
    TAG="$(date +%Y%m%d-%H%M%S)"
  fi
fi

tags=("$TAG")
if [[ "$NO_LATEST" != "1" ]]; then
  tags+=("latest")
fi

prefix="$REGISTRY"
if [[ -n "$NAMESPACE" ]]; then
  prefix="$REGISTRY/$NAMESPACE"
fi

web_image="$prefix/nocwall-ce-web"
api_image="$prefix/nocwall-ce-api"

build_and_push() {
  local image_base="$1"
  local dockerfile="$2"

  local cmd=(docker buildx build --platform "$PLATFORMS" -f "$dockerfile")
  for t in "${tags[@]}"; do
    cmd+=(-t "${image_base}:${t}")
  done
  cmd+=(--push)
  [[ "$PULL" == "1" ]] && cmd+=(--pull)
  [[ "$NO_CACHE" == "1" ]] && cmd+=(--no-cache)
  cmd+=(.)

  echo
  echo "Publishing image: $image_base"
  echo "Command: ${cmd[*]}"

  if [[ "$DRY_RUN" != "1" ]]; then
    "${cmd[@]}"
  fi
}

echo "Registry:  $REGISTRY"
echo "Namespace: $NAMESPACE"
echo "Platforms: $PLATFORMS"
echo "Tags:      ${tags[*]}"
echo "Builder:   $BUILDER"
if [[ "$DRY_RUN" == "1" ]]; then
  echo "DryRun:    enabled (no push will be performed)"
fi

build_and_push "$web_image" "Dockerfile"
build_and_push "$api_image" "api/Dockerfile"

echo
echo "Publish complete."
echo "Web image: $web_image"
echo "API image: $api_image"
echo "Tags: ${tags[*]}"
