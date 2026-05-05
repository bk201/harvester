# Migrate Dapper Script to Single-Dockerfile Build System

You are helping migrate a Harvester build target from the old Dapper (`scripts/` + `Dockerfile.dapper`) style to the new single-`Dockerfile` build system.

## Context

The new system uses one `Dockerfile` at the repo root with one stage per build target, driven directly from the `Makefile` via `docker build --target <stage>`. There are no separate `mk/<target>/docker-build` shell scripts.

### New system conventions (MUST follow exactly)

**Dockerfile structure**

- One `Dockerfile` at repo root; add a new stage per target — do not create per-target Dockerfiles
- First two lines of the Dockerfile:
  ```
  # syntax=docker/dockerfile:1
  # check=skip=InvalidDefaultArgInFrom
  ```
- Declare base image ARGs **before the first `FROM`**, then immediately alias the injected image as a named `builder` stage so other stages can reference it without repeating the ARG:
  ```dockerfile
  ARG MK_BUILDER_IMAGE
  ARG MK_BUNDLE_BUILDER_IMAGE
  FROM ${MK_BUILDER_IMAGE} AS builder
  ```
- `WORKDIR` and `ENV HOME` belong in `Dockerfile.builder`, baked into the builder image and inherited by all stages. `ENV HOME` must equal `WORKDIR` so `$HOME/.cache/go-build` resolves to the Go build cache mount path (mirrors Dapper's `ENV HOME ${DAPPER_SOURCE}`):
  ```dockerfile
  # in Dockerfile.builder — at the end, after tool installation:
  WORKDIR /go/src/github.com/harvester/harvester
  ENV HOME=/go/src/github.com/harvester/harvester
  ```
- `base` stage then only needs to copy the repo:
  ```dockerfile
  # ---- base ----
  FROM builder AS base
  COPY . .
  ```
- Because `.git` is present, git works natively inside all stages — no `git init` trick needed (relevant for `validate-ci` dirty check and `generate-openapi` vendor restore)
- **Exception — git worktree checkouts**: in a worktree, `.git` is a pointer *file* (e.g. `gitdir: ../.git/worktrees/foo`) whose target is outside the Docker build context. Git commands inside the container fail silently or error. See **Version scripts with git dependency** for the fix.
- Re-declare any `ARG` **inside each stage** that uses it (ARGs don't carry between stages):
  ```dockerfile
  FROM base AS build
  ARG MK_REPO_ID
  RUN ...
  ```
- Scope BuildKit cache mounts per repo: `id=harvester-go-mod-${MK_REPO_ID}`, `id=harvester-go-build-${MK_REPO_ID}`
- Standard cache-mount pattern for Go stages:
  ```dockerfile
  RUN --mount=type=cache,target=/go/pkg/mod,id=harvester-go-mod-${MK_REPO_ID} \
      --mount=type=cache,target=/go/src/github.com/harvester/harvester/.cache/go-build,id=harvester-go-build-${MK_REPO_ID} \
      ./scripts/<name>
  ```
- `generate-manifest` does not need Go cache mounts (only runs `controller-gen`, no compilation)
- The two `--mount=type=cache` lines repeat across every Go stage — this is unavoidable. Dockerfile has no macro or variable system for `RUN` option strings; `ARG`/`ENV` only substitute into specific fields (e.g. the `id=` value) and cannot expand into full `--mount` flags.

**File-extraction stages**

For targets that produce output files (binaries, generated files), add a companion `FROM scratch AS <stage>-output` stage:

```dockerfile
FROM scratch AS build-output
COPY --from=build /go/src/github.com/harvester/harvester/bin/ /bin/
```

The Makefile uses `--output type=local,dest=$(ROOT)` to extract them without `docker cp`. Docker maps the scratch image root to `dest`, so `/bin/` in the image becomes `$(ROOT)/bin/` on the host:

```makefile
build: builder-image
    $(BANNER)
    $(DOCKER_BUILD) --target build-output \
        --output type=local,dest=$(ROOT)
```

**DinD targets (need Docker socket at runtime)**

Targets like `test-integration` and `build-iso` can't access the Docker socket during `docker build`. Pattern:

1. Add an **empty** Dockerfile stage (just `FROM base AS <stage>`) — this creates the image with source code and tools
2. In the Makefile: build the image, then `docker run --privileged --network host -v /var/run/docker.sock:/var/run/docker.sock`
3. For Go-based integration tests, pass a named Docker volume for the Go build cache so repeated runs are fast:
   `-v harvester-<target>-go-cache-${MK_REPO_ID}:/go/src/github.com/harvester/harvester/.cache/go-build`

```dockerfile
# ---- test-integration ----
FROM base AS test-integration
```

```makefile
test-integration: builder-image
    $(DOCKER_BUILD) --target test-integration -t harvester-test-integration:$(MK_REPO_ID)
    docker run --rm --privileged --network host \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -v harvester-test-integration-go-cache-$(MK_REPO_ID):/go/src/github.com/harvester/harvester/.cache/go-build \
        harvester-test-integration:$(MK_REPO_ID) \
        ./scripts/test-integration
```

**Host-side targets (package / image push)**

Scripts that only call `docker buildx build` or `docker push` — and don't need the repo's Go toolchain or sources inside a container — run directly on the host. No Dockerfile stage needed.

```makefile
# ---- package ----
package: build
    $(BANNER)
    ARCH=$(HOST_ARCH) $(ROOT)/scripts/package
```

Pass `ARCH=$(HOST_ARCH)` inline so the script gets the correct architecture without relying on env exports. Optional caller-supplied vars (`REPO`, `TAG`) are picked up from the environment automatically since the script already defaults them.

If the package Dockerfiles need files that are excluded by `.dockerignore` (e.g. `bin/`), pass them as a named `--build-context` instead of modifying `.dockerignore`. Named build contexts bypass `.dockerignore` entirely:

```bash
# bin/ is excluded from the main build context by .dockerignore; pass it as a
# named build context so the package Dockerfiles can reach it via COPY --from=bin.
docker buildx build --load --build-context bin=./bin -f package/Dockerfile -t ${IMAGE} .
```

In the Dockerfile, drop the directory prefix and use `--from=<name>`:
```dockerfile
# instead of: COPY bin/harvester-network-controller-${ARCH} /usr/bin/...
COPY --from=bin harvester-network-controller-${ARCH} /usr/bin/harvester-network-controller
```

**`Dockerfile.builder` — separate file for the builder image**

The builder image lives in `Dockerfile.builder` at the repo root (not as a stage inside `Dockerfile`). This keeps the main `Dockerfile` free of base-image installation logic and lets `builder-image` build faster with its own layer cache.

`Dockerfile.builder` is a plain single-stage Dockerfile:
```dockerfile
# syntax=docker/dockerfile:1

FROM registry.suse.com/bci/golang:1.25.7

RUN zypper -n rm container-suseconnect && \
    zypper -n install git curl docker gzip tar wget awk
...
```

The `builder-image` Makefile target uses `-f $(ROOT)/Dockerfile.builder`:
```makefile
builder-image:
    docker build \
        --progress=$(MK_DOCKER_PROGRESS) \
        -t $(MK_BUILDER_IMAGE) \
        -f $(ROOT)/Dockerfile.builder $(ROOT)
```

**Bundle builder (separate base image)**

`build-bundle` uses `MK_BUNDLE_BUILDER_IMAGE` (not `MK_BUILDER_IMAGE`). Declare its ARG before the first `FROM` alongside `MK_BUILDER_IMAGE`:

```dockerfile
ARG MK_BUILDER_IMAGE
ARG MK_BUNDLE_BUILDER_IMAGE
FROM ${MK_BUILDER_IMAGE} AS builder
...
FROM ${MK_BUNDLE_BUILDER_IMAGE} AS build-bundle
```

The `bundle-builder` stage (built from `MK_BUILDER_IMAGE`) is tagged as `MK_BUNDLE_BUILDER_IMAGE` by `make bundle-builder-image`.

**Makefile conventions**

- Colors are defined conditionally so CI environments (which don't render ANSI) get plain output:
  ```makefile
  ifdef CI
    BOLD  :=
    CYAN  :=
    RESET :=
  else
    BOLD  := \033[1m
    CYAN  := \033[36m
    RESET := \033[0m
  endif
  ```
- `BANNER` prints the current target name using `$@` — use it as the first line of every recipe:
  ```makefile
  BANNER = @printf "$(BOLD)$(CYAN)[target: $@]$(RESET)\n"
  ```
- `MK_REPO_ID` must be exported so Make propagates it to any `scripts/mk-*` subprocesses — add `export MK_REPO_ID` immediately after the variable definition:
  ```makefile
  MK_REPO_ID := $(shell echo -n "$(ROOT)$$(cat /etc/machine-id 2>/dev/null)" | sha256sum | cut -c1-8)
  export MK_REPO_ID
  ```
- `MK_DOCKER_PROGRESS` controls BuildKit progress output and must default to `plain` (not `auto`). `plain` is required for CI legibility and for local terminals that don't support TTY progress rendering:
  ```makefile
  MK_DOCKER_PROGRESS ?= plain
  export MK_DOCKER_PROGRESS
  ```
- `DOCKER_BUILD` variable holds common flags:
  ```makefile
  DOCKER_BUILD = docker build \
      --progress=$(MK_DOCKER_PROGRESS) \
      --build-arg MK_BUILDER_IMAGE \
      --build-arg MK_REPO_ID \
      -f $(ROOT)/Dockerfile $(ROOT)
  ```
- Use `$(DOCKER_BUILD)` for all builder-image-based targets; write docker build inline only for targets with a different base (e.g. `build-bundle`)
- Extra per-target args append after `$(DOCKER_BUILD)`:
  ```makefile
  build-installer: builder-image | $(ROOT)/bin
      $(DOCKER_BUILD) --target build-installer-output \
          --build-arg HARVESTER_ADDONS_VERSION=$(HARVESTER_ADDONS_VERSION) \
          --output type=local,dest=$(ROOT)/bin
  ```
- When the `docker run` for a DinD target needs conditional env var passthrough (e.g. optional overrides), extract it to `scripts/mk-<target>` and call that from the Makefile:
  ```makefile
  build-iso: builder-image
      $(DOCKER_BUILD) --target build-iso -t harvester-iso-builder:$(MK_REPO_ID)
      $(ROOT)/scripts/mk-build-iso
  ```
  In `scripts/mk-<target>`, use bash `${VAR:+--env VAR="$VAR"}` for optional vars (not Makefile `$(if ...)`):
  ```bash
  docker run --rm --privileged \
      -v /var/run/docker.sock:/var/run/docker.sock \
      ${HARVESTER_INSTALLER_REF:+--env HARVESTER_INSTALLER_REF="$HARVESTER_INSTALLER_REF"} \
      "harvester-iso-builder:${MK_REPO_ID}" \
      ./scripts/build-iso
  ```
- When passing `--build-arg` in shell scripts, always use `--build-arg KEY="$VALUE"` (explicit value). `--build-arg KEY` (no `=`) reads from the **environment**, not from shell variables — if the variable isn't exported, Docker silently uses the `ARG` default from the Dockerfile.
- Add `HARVESTER_ADDONS_VERSION ?= main` for targets that clone addons
- `scripts/version` runs on the **host** (sourced by package scripts), so it must not use `go env GOHOSTARCH` — the host may not have Go installed. Use `uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/'` instead, which matches the `HOST_ARCH` mapping in the Makefile
- If the project has a `scripts/version` file that derives the entire version string from git (no static base version exists), see **Version scripts with git dependency** below — additional changes are required to support git worktree checkouts and container builds without `.git`

**Variable inventory (what to keep)**

Keep: `MK_REPO_ID`, `MK_BUILDER_IMAGE`, `MK_BUNDLE_BUILDER_IMAGE`, `MK_BUNDLE_IMAGE`, `MK_DOCKER_PROGRESS`, `HARVESTER_ADDONS_VERSION`, `MK_HARVESTER_INSTALLER_REPO`, `MK_HARVESTER_INSTALLER_REF`, `MK_RKE2_IMAGE_REPO`, `MK_USE_LOCAL_IMAGES`, `MK_REPO`, `HOST_ARCH`

Remove: `MK_DIR`, `MK_BIN_IMAGE`, `MK_INSTALLER_BIN_IMAGE`, `MK_ADDONS_IMAGE`, `MK_CONTAINER_WORKDIR`, `MK_ENV_FILE`, `MK_ENV_FILE_NAME`, `MK_DOCKER_BUILD_NO_CACHE`, `MK_ROOT`

Remove targets: `env`, `$(MK_ENV_FILE)`, `pull-addons`

Set `.DEFAULT_GOAL` to match the `CMD` in `Dockerfile.dapper` (the default Dapper entry point). Place it just before `.PHONY`:
```makefile
.DEFAULT_GOAL := ci

.PHONY: build builder-image ci ...
```

## Version scripts with git dependency

Some projects have a `scripts/version` file that is `source`d by multiple build scripts to set version variables (`VERSION`, `TAG`, `COMMIT`, `COMMIT_BRANCH`, etc.) entirely from git — there is no static base version hardcoded anywhere else.

This breaks in two scenarios:
- **Git worktree checkouts**: `.git` is a pointer *file* whose target resolves outside the Docker build context; git commands inside the container fail.
- **Container builds without `.git`**: if `.git` is excluded from the build context for any reason.

### Detection

All three signals must be present:
1. A script (typically `scripts/version`) calls `git rev-parse`, `git tag`, `git status`, or `git describe` and exports the results as shell variables
2. Other build scripts `source` it: `source $(dirname $0)/version`, `source $SCRIPTS_DIR/version`, etc.
3. No static `VERSION=x.y.z` assignment exists — the entire version string comes from git

### Changes to `scripts/version`

**At the top** — add a git availability guard before any `git` call. Declare the env file path using `TOP_DIR` (which the script already computes):

```bash
_VERSION_ENV="${TOP_DIR}/scripts/.version_env"

# When .git is not available (container build or git worktree checkout where the
# linked git dir is outside the Docker context), fall back to a pre-generated env
# file.  Run 'scripts/version' or 'make gen-version-env' on a host with git access.
if ! git -C "${TOP_DIR}" rev-parse HEAD &>/dev/null 2>&1; then
    if [[ -f "${_VERSION_ENV}" ]]; then
        # shellcheck source=/dev/null
        source "${_VERSION_ENV}"
        return 0 2>/dev/null || true
    fi
    echo "ERROR: git is unavailable and no pre-generated scripts/.version_env was found." >&2
    echo "Run 'scripts/version' (or 'make gen-version-env') on a host with git access first." >&2
    return 1 2>/dev/null || exit 1
fi
```

**At the bottom** — after all variables are computed, write them to `.version_env`. This runs automatically on every normal host invocation; no extra step is needed for local builds:

```bash
cat > "${_VERSION_ENV}" <<__EOF__
# Auto-generated by scripts/version — do not edit manually.
# Refresh by running scripts/version (or 'make gen-version-env') on a host with git access.
ARCH="${ARCH}"
DIRTY="${DIRTY}"
COMMIT="${COMMIT}"
COMMIT_BRANCH="${COMMIT_BRANCH}"
COMMIT_BRANCH_FORMATTED="${COMMIT_BRANCH_FORMATTED}"
GIT_TAG="${GIT_TAG}"
VERSION="${VERSION}"
IMAGE_PUSH_TAG="${IMAGE_PUSH_TAG}"
APP_VERSION="${APP_VERSION}"
CHART_VERSION="${CHART_VERSION}"
SUFFIX="${SUFFIX}"
TAG="${TAG}"
REPO="${REPO}"
# include every variable that callers read after sourcing scripts/version
__EOF__
```

### Remove redundant git calls in build scripts

If any build script re-runs git commands *after* sourcing `scripts/version` to recompute variables that `scripts/version` already set, remove those calls — they will fail inside the container:

```bash
# BEFORE — breaks inside container (git unavailable):
COMMIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
COMMIT_TAG=$(git tag --points-at HEAD | head -n 1)
if [[ "$COMMIT_TAG" == "" ]] && [["$COMMIT_BRANCH" == master || ...]]

# AFTER — reuse variables already set by scripts/version:
# COMMIT_BRANCH and GIT_TAG are already set by scripts/version above
if [[ "$GIT_TAG" == "" ]] && [[ "$COMMIT_BRANCH" == master || ...]]
```

### Makefile: `gen-version-env` target

Add a dedicated target and make it a prerequisite of **every** target that calls `$(DOCKER_BUILD)`:

```makefile
.PHONY: ... gen-version-env

# ---- Pre-generate version env for container builds (no .git needed inside Docker) ----
# Also handles git worktree checkouts where .git is a pointer file to an external directory.
gen-version-env:
	@bash $(ROOT)/scripts/version > /dev/null

build: builder-image gen-version-env | $(ROOT)/bin
validate: builder-image gen-version-env
validate-ci: builder-image gen-version-env
test: builder-image gen-version-env
test-integration: builder-image gen-version-env
generate-manifest: builder-image gen-version-env
generate-openapi: builder-image gen-version-env
build-iso: builder-image gen-version-env
# ... apply to all targets that invoke $(DOCKER_BUILD)
```

### `.gitignore` and `.dockerignore`

- Add `scripts/.version_env` to **`.gitignore`** — it is a generated artifact
- Do **not** add it to `.dockerignore` — the container must be able to read it

### Note on `validate-ci`

`validate-ci` checks for uncommitted changes introduced by `go generate` / `go mod tidy`. In a git worktree checkout `DIRTY` comes from the pre-generated `.version_env` (reflecting the state when `gen-version-env` last ran on the host), not a live `git status` inside the container. This is an inherent limitation of the worktree scenario, not a regression introduced by this change.

## Migration Procedure

Given a target name, follow these steps:

### Step 0 — Identify script type: delete or convert

First determine whether the script is a **Dapper orchestration script** or a **build script**:

**Delete (do not convert)** — Dapper entry/orchestration scripts that just sequence other scripts:
- Pattern: `cd $(dirname $0)` followed by calls to sibling scripts (`./build`, `./test`, `./package`, etc.)
- Examples removed during migration: `scripts/ci`, `scripts/arm`, `scripts/default`, `scripts/entry`, `scripts/generate`, `scripts/help`, `scripts/release`
- These are replaced entirely by the Makefile targets (`make ci`, `make arm`, etc.)

**Convert** — scripts that perform actual work and run inside a container:
- Pattern: run Go toolchain, git, docker, or generate files
- Examples: `scripts/build`, `scripts/validate`, `scripts/test`, `scripts/build-iso`, `scripts/generate-manifest`, `scripts/generate-openapi`

If the script is an orchestration script, delete it and ensure the corresponding Makefile target exists. No Dockerfile stage needed.

### Step 1 — Analyse the source script

Read `scripts/<name>` and identify:
1. **Type**: Go compile, validation, test, code generation, DinD (needs docker socket), file extraction
2. **Inputs**: env vars, files read, external repos cloned
3. **Outputs**: files written, directories populated, docker images produced
4. **Go cache needed**: does it run `go build` / `go test` / `go run`?
5. **Git needed**: does it call `git checkout`, `git status`, etc.? Normally fine — `.git` is copied in. **Exception for git worktrees**: if the script sources a `scripts/version`-style file that runs git to derive the entire version string, apply the version-env fallback (see **Version scripts with git dependency**)

### Step 2 — Choose stage type

| Pattern | When to use |
|---|---|
| `FROM base AS <name>` + `RUN ./scripts/<name>` | Standard compile/validate/gen (no Docker needed at runtime) |
| `FROM scratch AS <name>-output` + `COPY --from=<name>` | When the target produces files to extract to host |
| Empty `FROM base AS <name>` + DinD `docker run` | Target needs Docker socket (test-integration, build-iso) |
| `FROM ${MK_BUNDLE_BUILDER_IMAGE} AS <name>` | build-bundle (different base image) |

### Step 3 — Add stage to Dockerfile

Insert the new stage(s) at the appropriate location in `/Dockerfile`. Keep stages grouped by type (compile, validate, test, generate, build-installer, build-iso, bundle). Add a comment header:

```dockerfile
# ---- <target-name> ----
FROM base AS <target-name>
ARG MK_REPO_ID

RUN --mount=type=cache,target=/go/pkg/mod,id=harvester-go-mod-${MK_REPO_ID} \
    --mount=type=cache,target=/go/src/github.com/harvester/harvester/.cache/go-build,id=harvester-go-build-${MK_REPO_ID} \
    ./scripts/<name>
```

If the stage produces output files, add the scratch stage immediately after:

```dockerfile
FROM scratch AS <target-name>-output
COPY --from=<target-name> /go/src/github.com/harvester/harvester/<output-path> /<dest-path>
```

### Step 4 — Add Makefile target

Add the target to `.PHONY` and write the rule. Use `$(DOCKER_BUILD)`.

If the project uses the **version-env pattern** (see **Version scripts with git dependency**), add `gen-version-env` as a prerequisite to every target that calls `$(DOCKER_BUILD)`:

```makefile
# ---- Target description ----
<target-name>: builder-image gen-version-env
	$(BANNER)
	$(DOCKER_BUILD) --target <target-name>
```

For extraction targets:
```makefile
<target-name>: builder-image gen-version-env
	$(BANNER)
	$(DOCKER_BUILD) --target <target-name>-output --output type=local,dest=$(ROOT)/<path>
```

For simple DinD targets (no conditional env vars):
```makefile
<target-name>: builder-image gen-version-env
	$(BANNER)
	$(DOCKER_BUILD) --target <target-name> -t harvester-<target-name>:$(MK_REPO_ID)
	docker run --rm --privileged --network host \
	    -v /var/run/docker.sock:/var/run/docker.sock \
	    harvester-<target-name>:$(MK_REPO_ID) \
	    ./scripts/<name>
```

For Go-based DinD targets (e.g. integration tests), also pass a named Go build cache volume:
```makefile
	docker run --rm --privileged --network host \
	    -v /var/run/docker.sock:/var/run/docker.sock \
	    -v harvester-<target-name>-go-cache-$(MK_REPO_ID):/go/src/github.com/harvester/harvester/.cache/go-build \
	    harvester-<target-name>:$(MK_REPO_ID) \
	    ./scripts/<name>
```

For DinD targets with conditional env var passthrough, extract to `scripts/mk-<target-name>` and validate `MK_REPO_ID` at the top:
```makefile
<target-name>: builder-image
	$(BANNER)
	$(DOCKER_BUILD) --target <target-name> -t harvester-<target-name>:$(MK_REPO_ID)
	$(ROOT)/scripts/mk-<target-name>
```
```bash
#!/bin/bash
set -e
TOP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
MK_REPO_ID="${MK_REPO_ID:?MK_REPO_ID is required}"

docker run --rm --privileged --network host \
    -v /var/run/docker.sock:/var/run/docker.sock \
    ${OPTIONAL_VAR:+--env OPTIONAL_VAR="$OPTIONAL_VAR"} \
    "harvester-<target-name>:${MK_REPO_ID}" \
    ./scripts/<name>
```

### Step 5 — Ensure `.dockerignore` exists

If `.dockerignore` does not exist at the repo root, seed it from `.gitignore` (which already captures build artifacts, IDE dirs, and OS noise), then ensure `/bin` and `/dist` are present:

```bash
cp .gitignore .dockerignore
# then add any missing entries:
grep -qxF '/dist' .dockerignore || echo '/dist' >> .dockerignore
```

**Do NOT add `/bin` to `.dockerignore`** if any host-side script (e.g. `scripts/package`) runs `docker buildx build` with the repo root as build context and the package Dockerfiles do `COPY bin/...`. Excluding `/bin` would make those binaries invisible to the build. Instead, add a comment explaining the intent:

```
# /bin is not ignored because `make package` uses the project root as docker build context
```

Note: `.git` is intentionally **not** excluded — the `base` stage needs it for dirty-check targets. `.gitignore` never lists `.git`, so copying it is safe. In git worktree checkouts the `.git` pointer file is copied but the object database is unreachable inside Docker; `scripts/.version_env` provides the fallback (see **Version scripts with git dependency**).

### Step 6 — Update clean/clean-all

- Add `@rm -rf` / `@rm -f` to `clean` for any files/dirs the target extracts to the host
- Add `@docker rmi -f harvester-<target-name>:$(MK_REPO_ID) || true` to `clean-all` for DinD images

### Step 7 — Validate

1. Each `ARG` used in a stage body is declared **inside** that stage (not only before first FROM)
2. `--output type=local` destination matches the COPY paths in the scratch stage
3. DinD targets mount the docker socket and use `--privileged`
4. `.PHONY` list includes the new target

## Reference: current Dockerfile stages

| Stage | Type | Output stage? | Notes |
|---|---|---|---|
| `base` | base | — | `COPY . .` includes `.git` |
| `build` | compile | `build-output` | harvester, webhook, upgrade-helper binaries |
| `validate` | lint | — | `./scripts/validate` |
| `validate-ci` | dirty check | — | `./scripts/validate-ci`; git works because `.git` copied |
| `test` | unit test | — | `./scripts/test` |
| `test-integration` | DinD | — | empty stage; docker run runs `./scripts/test-integration` |
| `generate-manifest` | codegen | `generate-manifest-output` | no Go cache needed |
| `generate-openapi` | codegen | `generate-openapi-output` | git works for `git checkout -- vendor` |
| `build-installer` | compile | `build-installer-output` | clones addons; `ENV GOFLAGS=-buildvcs=false` |
| `build-iso` | DinD | — | empty stage; docker run runs `./scripts/build-iso` |
| `bundle-builder` | base | — | tagged as `MK_BUNDLE_BUILDER_IMAGE` |
| `build-bundle` | bundle | — | `FROM ${MK_BUNDLE_BUILDER_IMAGE}`; separate docker build call |

## GitHub Actions artifact passing

When a build stage produces output that must be passed between CI jobs (e.g. `prepare-addons` manifests), follow this pattern:

**Upload** — upload the directory path directly:
```yaml
- uses: actions/upload-artifact@v4
  with:
    name: addons_${{ matrix.arch }}_artifact
    path: ./dist/prepare-addons/addons-manifests
```
GitHub uploads the **contents** of the directory, not the directory itself.

**Download + place** — download to the exact final path the consumer expects, avoiding an intermediate `cp`:
```yaml
- uses: actions/download-artifact@v4
  with:
    name: addons_${{ matrix.arch }}_artifact
    path: ./package/upgrade/addons      # ← the name the Dockerfile's COPY expects
```

**Do NOT** download to an intermediate path and then `cp -r <intermediate> <dest>/` — this creates `<dest>/<intermediate-dirname>/` (one level too deep) instead of placing files directly in `<dest>/`.

**Do NOT** call `make generate-addons` (or any re-generation step) after downloading the artifact — it will overwrite the pinned artifact with a fresh clone that may be at a different commit.

## Usage

### Single target

The user names one target (e.g. "migrate generate-addons"). Read `scripts/<name>`, execute Steps 0–7 for that target, then summarise: Dockerfile changes, Makefile changes, and any caveats.

### All targets (`migrate all`)

When the user says "migrate all" or "migrate all targets":

1. **Inventory** — list every file in `scripts/`. Read each one.
2. **Classify** — apply Step 0 to every script: mark each as **delete** (orchestration) or **convert** (real work).
3. **Skip already-migrated** — check the existing `Dockerfile` for stage names and the `Makefile` for target rules; skip any script whose stage/target is already present.
4. **Plan in dependency order** — scripts that produce outputs consumed by others (e.g. `build` before `package`) must be migrated first. Resolve the order before writing anything.
5. **Execute Steps 1–7 for each remaining convert script** in one pass:
   - Append all new stages to `Dockerfile` in a single edit (grouped by type: compile → validate → test → generate → package/DinD)
   - Add all new Makefile targets in a single edit
   - Update `.PHONY`, `clean`, and `clean-all` once at the end
6. **Summarise** in a table: each script → action taken (deleted / stage added / host target / skipped).
