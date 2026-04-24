# syntax=docker/dockerfile:1
# check=skip=InvalidDefaultArgInFrom
ARG MK_BUILDER_IMAGE
FROM ${MK_BUILDER_IMAGE} AS builder

ARG MK_CONTAINER_WORKDIR=/go/src/github.com/harvester/harvester
ARG MK_REPO_ID
ARG MK_ENV_FILE_NAME=mk-env.sh
WORKDIR ${MK_CONTAINER_WORKDIR}

COPY go.mod go.sum go.work go.work.sum ./
COPY installer/ installer/
COPY vendor/ vendor/
COPY main.go ./
COPY pkg/ pkg/
COPY scripts/ scripts/
COPY deploy/ deploy/
COPY package/ package/

COPY ${MK_ENV_FILE_NAME} mk-env.sh

COPY mk/build-iso/build mk/build-iso/build
