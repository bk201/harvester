FROM harvester-builder:local

ARG CONTAINER_WORKDIR=/go/src/github.com/harvester/harvester
WORKDIR ${CONTAINER_WORKDIR}

# Vendor and module files change rarely — keep as early cached layers
COPY go.mod go.sum go.work go.work.sum ./
COPY installer/go.mod installer/go.sum installer/

# TODO: installer lint

# Source code
COPY main.go ./
COPY pkg/ pkg/
COPY cmd/ cmd/
COPY tests/ tests/
COPY .golangci.yaml .golangci.yaml

RUN --mount=type=cache,target=/go/pkg/mod,id=harvester-go-mod \
    --mount=type=cache,target=${CONTAINER_WORKDIR}/.cache/go-build,id=harvester-go-build \
    golangci-lint run -v --timeout=20m
