FROM harvester-bundle-builder:local

ARG CONTAINER_WORKDIR=/workspace
WORKDIR ${CONTAINER_WORKDIR}

# Copy necessary source files
COPY deploy/ harvester/deploy/
COPY installer/ installer/
COPY scripts/version harvester/scripts/version
COPY harvester-env.sh ./harvester-env.sh

# Run build-bundle script with Docker socket access via BuildKit mount
# The script will source harvester-env.sh to get version variables
RUN cd installer && \
    HARVESTER_DIR=/workspace/harvester ./scripts/build-bundle
