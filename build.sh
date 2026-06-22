#!/usr/bin/env bash
# Build the server image, sourcing the Minecraft/Fabric versions from
# manifest.lock so they live in exactly one place. Extra args are passed through
# to `docker buildx build`, e.g.:
#
#   ./build.sh --load -t ghcr.io/arntio/mc-smp:dev .
#   ./build.sh --push -t ghcr.io/arntio/mc-smp:v2026.06.22 -t ghcr.io/arntio/mc-smp:latest .
set -euo pipefail
cd "$(dirname "$0")"

# Pull a scalar value out of manifest.lock (simple, fixed structure).
val() { grep -E "$1" manifest.lock | head -1 | sed -E 's/^[^:]*:[[:space:]]*"?([^"[:space:]]+)"?.*/\1/'; }
VERSION="$(val '^minecraft:')"
FABRIC_LOADER_VERSION="$(val '^[[:space:]]+loader:')"
FABRIC_LAUNCHER_VERSION="$(val '^[[:space:]]+installer:')"

: "${VERSION:?could not read minecraft version from manifest.lock}"
: "${FABRIC_LOADER_VERSION:?could not read fabric loader from manifest.lock}"
: "${FABRIC_LAUNCHER_VERSION:?could not read fabric installer from manifest.lock}"

echo "Building MC $VERSION / fabric loader $FABRIC_LOADER_VERSION / installer $FABRIC_LAUNCHER_VERSION"
exec docker buildx build \
  --build-arg "VERSION=$VERSION" \
  --build-arg "FABRIC_LOADER_VERSION=$FABRIC_LOADER_VERSION" \
  --build-arg "FABRIC_LAUNCHER_VERSION=$FABRIC_LAUNCHER_VERSION" \
  "$@"
