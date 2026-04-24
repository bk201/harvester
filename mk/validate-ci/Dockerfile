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
COPY cmd/ cmd/
COPY tests/ tests/

# For scripts/boilerplate.go.txt, swagger.json
COPY scripts/ scripts/
COPY api/ api/

COPY mk/validate-ci/build mk/validate-ci/build

FROM builder AS run

ARG MK_CONTAINER_WORKDIR
ARG MK_REPO_ID

# Init a new git repo for copied files inside the container, the test script checks if files are
# modified after running go generate
RUN git config --global user.email "ci@example.com" && \
    git config --global user.name "ci" && \
    git init 2>/dev/null && git add . && git commit -q -m "commit for validate-ci"

RUN --mount=type=cache,target=/go/pkg/mod,id=harvester-go-mod-${MK_REPO_ID} \
    --mount=type=cache,target=${MK_CONTAINER_WORKDIR}/.cache/go-build,id=harvester-go-build-${MK_REPO_ID} \
    bash mk/validate-ci/build
