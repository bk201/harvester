# syntax=docker/dockerfile:1
# check=skip=InvalidDefaultArgInFrom
ARG MK_BUILDER_IMAGE
FROM ${MK_BUILDER_IMAGE} AS builder

ARG MK_CONTAINER_WORKDIR=/go/src/github.com/harvester/harvester
WORKDIR ${MK_CONTAINER_WORKDIR}

COPY go.mod go.sum go.work go.work.sum ./
COPY installer/go.mod installer/go.sum installer/
COPY vendor/ vendor/

COPY main.go ./
COPY pkg/ pkg/
COPY api/ api/
COPY scripts/generate-openapi scripts/boilerplate.go.txt scripts/

FROM builder AS run

ARG MK_CONTAINER_WORKDIR
ARG MK_REPO_ID

# Init a git repo so scripts/generate-openapi can restore vendor doc.go files via git checkout
RUN git config --global user.email "ci@example.com" && \
    git config --global user.name "ci" && \
    git init 2>/dev/null && git add . && git commit -q -m "init"

RUN --mount=type=cache,target=/go/pkg/mod,id=harvester-go-mod-${MK_REPO_ID} \
    --mount=type=cache,target=${MK_CONTAINER_WORKDIR}/.cache/go-build,id=harvester-go-build-${MK_REPO_ID} \
    bash scripts/generate-openapi
