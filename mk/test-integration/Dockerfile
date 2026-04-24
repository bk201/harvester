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
COPY deploy/ deploy/

# Version variables file
ARG MK_ENV_FILE_NAME=mk-env.sh
COPY ${MK_ENV_FILE_NAME} mk-env.sh

COPY mk/test-integration/build mk/test-integration/build
