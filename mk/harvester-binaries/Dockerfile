# syntax=docker/dockerfile:1
# check=skip=InvalidDefaultArgInFrom
ARG MK_BUILDER_IMAGE
FROM ${MK_BUILDER_IMAGE} AS builder

ARG MK_CONTAINER_WORKDIR=/go/src/github.com/harvester/harvester
WORKDIR ${MK_CONTAINER_WORKDIR}

# Vendor and module files change rarely — keep as early cached layers
COPY vendor/ vendor/
COPY go.mod go.sum go.work go.work.sum ./
COPY installer/go.mod installer/go.sum installer/

# Source code
COPY main.go ./
COPY pkg/ pkg/
COPY cmd/ cmd/

# Version variables file
ARG MK_ENV_FILE_NAME=mk-env.sh
COPY ${MK_ENV_FILE_NAME} mk-env.sh

# Build script (will source mk-env.sh)
COPY mk/harvester-binaries/build mk/harvester-binaries/build

FROM builder AS run

ARG MK_CONTAINER_WORKDIR
ARG MK_REPO_ID
ENV GOFLAGS=-buildvcs=false

RUN --mount=type=cache,target=/go/pkg/mod,id=harvester-go-mod-${MK_REPO_ID} \
    --mount=type=cache,target=${MK_CONTAINER_WORKDIR}/.cache/go-build,id=harvester-go-build-${MK_REPO_ID} \
    bash mk/harvester-binaries/build
