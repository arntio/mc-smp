# syntax=docker/dockerfile:1

# ── Stage 1: resolve & download mods + datapacks from manifest.lock ──────────
FROM golang:1.25-alpine AS builder
WORKDIR /src
# Build the updater (deps first for layer caching).
COPY tools/go.mod tools/go.sum ./tools/
RUN cd tools && go mod download
COPY tools/ ./tools/
RUN cd tools && CGO_ENABLED=0 go build -o /usr/local/bin/updater .
# Download exactly what the lock pins (mods verified by sha512).
COPY manifest.lock ./
RUN updater download --lock manifest.lock --out /out

# ── Stage 2: the runnable server image ──────────────────────────────────────
FROM itzg/minecraft-server:stable

# Minecraft + Fabric versions are supplied at build time from manifest.lock
# (see build.sh) — nothing version-specific is hard-coded in this file.
ARG VERSION
ARG FABRIC_LOADER_VERSION
ARG FABRIC_LAUNCHER_VERSION
RUN test -n "$VERSION" && test -n "$FABRIC_LOADER_VERSION" && test -n "$FABRIC_LAUNCHER_VERSION" \
    || { echo "ERROR: build with ./build.sh, which sources versions from manifest.lock"; exit 1; }

# Baked, version-controlled content. itzg syncs these into /data on each start
# (REMOVE_OLD_* clears stale files), so a new image tag swaps mods/datapacks
# cleanly while the world/whitelist/bans persist in the /data volume.
COPY --from=builder /out/mods /image-mods
COPY --from=builder /out/datapacks /image-datapacks
COPY assets/ /assets/
# The updater binary doubles as the backup tool (used by the backup sidecar).
COPY --from=builder /usr/local/bin/updater /usr/local/bin/updater

# Image identity + build mechanics. These are intrinsic to this image and are
# NOT meant to be changed at deploy time. Tunable gameplay/runtime settings
# (memory, MOTD, difficulty, ...) live in compose + .env instead.
ENV VERSION="${VERSION}" \
    FABRIC_LOADER_VERSION="${FABRIC_LOADER_VERSION}" \
    FABRIC_LAUNCHER_VERSION="${FABRIC_LAUNCHER_VERSION}" \
    EULA="TRUE" \
    TYPE="FABRIC" \
    OVERRIDE_ICON="true" \
    ICON="/assets/snake_logo.png" \
    COPY_MODS_SRC="/image-mods" \
    REMOVE_OLD_MODS="true" \
    REMOVE_OLD_MODS_INCLUDE="*.jar" \
    REMOVE_OLD_MODS_DEPTH="1" \
    DATAPACKS="/image-datapacks" \
    REMOVE_OLD_DATAPACKS="true"
