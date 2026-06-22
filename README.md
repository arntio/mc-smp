# SMP — Minecraft Fabric Server

> [!NOTE]  
> Yes, the latest changes were vibe coded.

A self-contained, auto-updating Minecraft Fabric server. Mods, datapacks and
crafting tweaks are declared in [`manifest.yaml`](manifest.yaml); the build
downloads and bakes them into a single Docker image. Running or updating the
live server is just a `docker compose pull` — no `git pull`, no merge conflicts,
no manual jar wrangling.

## How it works

```
manifest.yaml   you edit this: Modrinth slugs + Vanilla Tweaks pack names
     │  updater (tools/) resolves newest compatible versions
     ▼
manifest.lock   generated: exact versions + URLs + sha512 (reproducible)
     │  docker build downloads & bakes mods + datapacks
     ▼
ghcr.io/arntio/mc-smp:<version>   standalone image, published on each release
```

- **Mods** come from [Modrinth](https://modrinth.com/) (pinned by version + sha512).
- **Datapacks & crafting tweaks** come from [Vanilla Tweaks](https://vanillatweaks.net/)
  via its API (pinned by name + version, re-fetched at build time).
- **Minecraft + Fabric** are downloaded by the [itzg/minecraft-server](https://docker-minecraft-server.readthedocs.io/)
  base image on first start (cached in the `data` volume afterwards).

On every start the image re-syncs mods/datapacks from the baked copy
(`REMOVE_OLD_MODS` / `REMOVE_OLD_DATAPACKS`), so pulling a new image swaps them
cleanly. `server.properties` is regenerated from the image's environment each
start (volatile), while `world`, `whitelist.json` and `banned-*.json` persist in
the `data` volume (permanent).

## Run the live server

Needs Docker + Docker Compose. On the host you only need `compose.yaml`, a
`.env`, and the `./data` directory.

```bash
cp .env.example .env        # fill in RCON_PASSWORD + R2 credentials
docker compose -f compose.yaml pull
docker compose -f compose.yaml up -d
```

The world and player data live in `./data` (created on first run). The first
start downloads Minecraft + Fabric and may take a couple of minutes.

> `pull` needs the image published to GHCR first — that happens automatically
> when changes land on `main` (Release → Publish image). For the very first
> deploy you can instead build it on the host:
> `./build.sh --load -t ghcr.io/arntio/mc-smp:latest .` then `up -d` (skip `pull`).

### Migrating an existing world

The world is whatever sits in `./data` — mods and datapacks are **not** stored
there (they come from the image). So migrating is just bringing the old world
across:

```bash
# on the host, next to compose.yaml
mkdir -p data
rsync -a /path/to/old-server/data/ ./data/     # world, whitelist.json, banned-*.json, ops.json, playerdata...
```

Then `up -d`. The image repopulates `data/mods` and `data/world/datapacks` from
its baked copies on start (old ones are cleared first), and `server.properties`
is regenerated from the env — so reuse the **same** world data, but you do not
need to copy the old `mods/` or `datapacks/` folders.

### Update to a newer version

```bash
docker compose -f compose.yaml pull
docker compose -f compose.yaml up -d
```

That's it. The world is untouched; only mods/datapacks/versions change.

## Change the mods or datapacks

Edit [`manifest.yaml`](manifest.yaml) (add/remove a Modrinth slug or a Vanilla
Tweaks pack name) and open a PR. Regenerate the lock locally first:

```bash
cd tools && go run . lock --root ..    # regenerates manifest.lock
```

The PR build (`Validate`) downloads everything and boots the server to prove it
works. Merge once it's green.

## Automation

| Workflow | Trigger | Does |
|----------|---------|------|
| `Check for updates` | weekly + manual | Re-locks to the newest compatible versions and opens a PR. Minecraft is only bumped when **all** mods have a compatible build. |
| `Validate` | PR to `main` | Builds the image and boots the server; fails on startup errors. Required to merge. |
| `Release` | push to `main` | Cuts a dated release (`vYYYY.MM.DD`) with a changelog. |
| `Publish image` | release published | Builds and pushes `ghcr.io/arntio/mc-smp:<version>` + `:latest`. |

## Backups

The `backup` service in `compose.yaml` runs the bundled `updater backup`
tool (same image, different entrypoint). It uploads a plain, unencrypted
`.tar.gz` of the world to a Cloudflare R2 bucket and:

- runs **daily** (`BACKUP_INTERVAL=24h`);
- **skips a cycle if no player was seen online** since the last backup (polls
  RCON every `BACKUP_PLAYER_POLL`) — idle periods produce no backups;
- **skips a cycle if the world is byte-for-byte unchanged** (a fingerprint of
  file sizes + mtimes is compared against the last backup);
- takes a **consistent** snapshot: `save-off` + `save-all flush` over RCON, then
  `save-on`;
- **keeps the last `BACKUP_KEEP` (7) backups**, pruning older ones.

Configure `R2_*` in `.env`. Tunables: `BACKUP_INTERVAL`, `BACKUP_KEEP`,
`BACKUP_PLAYER_POLL`, `BACKUP_PATHS`, `BACKUP_PREFIX`.

Manual / restore helpers:

```bash
# one-off backup now (bypassing the player/no-change checks)
docker compose -f compose.yaml run --rm backup backup --once --force

# restore: download a .tar.gz from R2 and extract into ./data
tar -xzf snake-smp-<timestamp>.tar.gz -C ./data
```

## Configuration

- **Baked into the image** (build/identity, not meant to change per-deploy):
  Minecraft + Fabric versions (from `manifest.lock`), mods, datapacks, icon,
  `EULA`, `TYPE`, the mod/datapack sync settings.
- **Tunable at deploy time** via `.env` (see `.env.example`): `INIT_MEMORY`,
  `MAX_MEMORY`, `MOTD`, `DIFFICULTY`, `ALLOW_FLIGHT`, `ENABLE_COMMAND_BLOCK`,
  `SPAWN_PROTECTION`, `VIEW_DISTANCE`, `SEED`, `ENABLE_WHITELIST`. Sensible
  defaults apply when unset.

## Local development

```bash
./build.sh --load -t ghcr.io/arntio/mc-smp:dev .   # download + bake from the lock
docker compose up                                  # run locally on port 25567
```

`build.sh` reads the Minecraft/Fabric versions from `manifest.lock` and passes
them as build args, so the versions live in exactly one place.

## One-time setup

- **Branch protection** on `main`: require the `Validate` check.
- **GHCR**: the publish workflow uses the built-in `GITHUB_TOKEN` (no secret to add).
- **Server host secrets** (`.env`): `RCON_PASSWORD`, `R2_ACCOUNT_ID`,
  `R2_BUCKET`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`.

## Layout

| Path | Purpose |
|------|---------|
| `manifest.yaml` | What to install (you edit this) |
| `manifest.lock` | Resolved pins (generated) |
| `Dockerfile` | Multi-stage build: download → bake (versions via build args) |
| `build.sh` | Builds the image, sourcing versions from `manifest.lock` |
| `tools/` | Go CLI: `lock` / `check` / `download` + `backup` |
| `compose.yaml` | Local dev/build |
| `compose.yaml` | Live server + backups |
| `.github/workflows/` | Update / validate / release / publish |
