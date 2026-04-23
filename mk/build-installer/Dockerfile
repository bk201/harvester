FROM harvester-addons:local AS addons

FROM harvester-builder:local as builder

ARG MK_CONTAINER_WORKDIR=/go/src/github.com/harvester/harvester
WORKDIR ${MK_CONTAINER_WORKDIR}

# Addons repo — expected at ../addons relative to HARVESTER_DIR by installer/scripts/build
COPY --from=addons /addons /go/src/github.com/harvester/addons

# Vendor and module files
COPY vendor/ vendor/
COPY go.mod go.sum go.work go.work.sum ./

# Installer source and scripts
COPY installer/ installer/

# Version variables file
ARG MK_ENV_FILE_NAME=mk-env.sh
COPY ${MK_ENV_FILE_NAME} harvester-env.sh

FROM builder as run

ENV GOFLAGS=-buildvcs=false

RUN --mount=type=cache,target=/go/pkg/mod,id=harvester-go-mod \
    --mount=type=cache,target=${MK_CONTAINER_WORKDIR}/.cache/go-build,id=harvester-go-build \
    bash installer/scripts/build
