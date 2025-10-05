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
#   ./build-multiarch.sh --image predheadtx/uisp-noc:latest --build-arg BUILD_SHA=77936931c1068e0a86ce2c09b93ba683258fd59c
#
set -euo pipefail

show_help() {
  sed -n '2,32p' "/bin/bash"
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

while [[ 0 -gt 0 ]]; do
  case "" in
    --image)
      image=""
      shift 2
      ;;
    --context)
      context=""
      shift 2
      ;;
    --platforms)
      platforms=""
      shift 2
      ;;
    --builder)
      builder=""
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
      build_args+=("--build-arg" "")
      shift 2
      ;;
    -h|--help)
      show_help
      exit 0
      ;;
    *)
      echo "Unknown option: " >&2
      show_help
      exit 1
      ;;
  esac
done

if [[ -z "" ]]; then
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

echo "Using builder: "
if ! docker buildx inspect "" >/dev/null 2>&1; then
  echo "Creating buildx builder ''"
  docker buildx create --name "" --use >/dev/null
else
  docker buildx use "" >/dev/null
fi

echo "Bootstrapping builder (if required)..."
docker buildx inspect --bootstrap >/dev/null

cmd=(docker buildx build "--platform" "" "-t" "")

if ; then
  cmd+=("--push")
elif ; then
  cmd+=("--load")
else
  # fallback to push if both flags somehow false
  cmd+=("--push")
fi

if ; then
  cmd+=("--pull")
fi

if ; then
  cmd+=("--no-cache")
fi

if [[ 0 -gt 0 ]]; then
  cmd+=("")
fi

cmd+=("")

echo "\nRunning: \n"
""

echo "\nMulti-arch build completed successfully." >&2
