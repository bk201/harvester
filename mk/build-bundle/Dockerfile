# syntax=docker/dockerfile:1
# check=skip=InvalidDefaultArgInFrom
ARG MK_BUNDLE_BUILDER_IMAGE
FROM ${MK_BUNDLE_BUILDER_IMAGE} AS builder

ARG MK_CONTAINER_WORKDIR=/workspace
WORKDIR ${MK_CONTAINER_WORKDIR}

# Copy necessary source files
COPY deploy/ harvester/deploy/
COPY installer/ installer/
COPY scripts/version harvester/scripts/version
ARG MK_ENV_FILE_NAME=mk-env.sh
COPY ${MK_ENV_FILE_NAME} harvester-env.sh

FROM builder AS run

RUN cd installer && \
    HARVESTER_DIR=/workspace/harvester ./scripts/build-bundle
