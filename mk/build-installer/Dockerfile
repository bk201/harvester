# syntax=docker/dockerfile:1
# check=skip=InvalidDefaultArgInFrom
ARG MK_ADDONS_IMAGE
ARG MK_BUILDER_IMAGE
FROM ${MK_ADDONS_IMAGE} AS addons
FROM ${MK_BUILDER_IMAGE} AS builder

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

FROM builder AS run

ARG MK_CONTAINER_WORKDIR
ARG MK_REPO_ID
ENV GOFLAGS=-buildvcs=false

RUN --mount=type=cache,target=/go/pkg/mod,id=harvester-go-mod-${MK_REPO_ID} \
    --mount=type=cache,target=${MK_CONTAINER_WORKDIR}/.cache/go-build,id=harvester-go-build-${MK_REPO_ID} \
    bash installer/scripts/build
