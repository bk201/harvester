FROM harvester-bundle-builder:local as builder

ARG CONTAINER_WORKDIR=/workspace
WORKDIR ${CONTAINER_WORKDIR}

# Copy necessary source files
COPY deploy/ harvester/deploy/
COPY installer/ installer/
COPY scripts/version harvester/scripts/version
COPY harvester-env.sh ./harvester-env.sh

FROM builder as run

RUN cd installer && \
    HARVESTER_DIR=/workspace/harvester ./scripts/build-bundle
