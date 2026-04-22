FROM harvester-builder:local

ARG CONTAINER_WORKDIR=/go/src/github.com/harvester/harvester
WORKDIR ${CONTAINER_WORKDIR}

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
COPY harvester-env.sh ./

COPY mk/test-integration/build mk/test-integration/build
