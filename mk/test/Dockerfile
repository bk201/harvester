FROM harvester-builder:local as builder

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

FROM builder as run

RUN --mount=type=cache,target=/go/pkg/mod,id=harvester-go-mod \
    --mount=type=cache,target=${MK_CONTAINER_WORKDIR}/.cache/go-build,id=harvester-go-build \
    bash mk/test/build
