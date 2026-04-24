# syntax=docker/dockerfile:1
# check=skip=InvalidDefaultArgInFrom
ARG MK_BUILDER_IMAGE
FROM ${MK_BUILDER_IMAGE} AS builder

ARG MK_CONTAINER_WORKDIR=/go/src/github.com/harvester/harvester
WORKDIR ${MK_CONTAINER_WORKDIR}

COPY go.mod go.sum go.work go.work.sum ./
COPY installer/go.mod installer/go.sum installer/
COPY vendor/ vendor/
COPY pkg/apis/ pkg/apis/
COPY scripts/generate-manifest scripts/generate-manifest

FROM builder AS run

RUN bash scripts/generate-manifest
