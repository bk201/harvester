ARG MK_BUILDER_IMAGE
FROM ${MK_BUILDER_IMAGE}

ARG HARVESTER_ADDONS_VERSION=main

RUN git clone --branch ${HARVESTER_ADDONS_VERSION} --single-branch --depth 1 \
    https://github.com/harvester/addons.git /addons && \
    rm -rf /addons/.git
