# syntax=docker/dockerfile:1
# check=skip=InvalidDefaultArgInFrom
ARG MK_BUILDER_IMAGE
FROM ${MK_BUILDER_IMAGE} AS builder

ARG MK_CONTAINER_WORKDIR=/go/src/github.com/harvester/harvester
WORKDIR ${MK_CONTAINER_WORKDIR}

# Vendor and module files change rarely — keep as early cached layers
COPY go.mod go.sum go.work go.work.sum ./
COPY installer/go.mod installer/go.sum installer/

# Source code
COPY main.go ./
COPY pkg/ pkg/
COPY cmd/ cmd/
COPY tests/ tests/

COPY mk/test/build mk/test/build

FROM builder AS run

ARG MK_CONTAINER_WORKDIR
ARG MK_REPO_ID
ARG MK_ENV_FILE_NAME

RUN --mount=type=cache,target=/go/pkg/mod,id=harvester-go-mod-${MK_REPO_ID} \
    --mount=type=cache,target=${MK_CONTAINER_WORKDIR}/.cache/go-build,id=harvester-go-build-${MK_REPO_ID} \
    bash mk/test/build
