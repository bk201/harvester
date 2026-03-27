ROOT              := $(realpath $(dir $(realpath $(firstword $(MAKEFILE_LIST)))))
MK_DIR            := $(ROOT)/mk
BUILDER_IMAGE          := harvester-builder:local
BIN_IMAGE              := harvester-bin:local
INSTALLER_BIN_IMAGE    := harvester-installer-bin:local
ADDONS_IMAGE           := harvester-addons:local
CONTAINER_WORKDIR := /go/src/github.com/harvester/harvester
ENV_FILE          := $(ROOT)/harvester-env.sh
HOST_ARCH         := $(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
DOCKER_PROGRESS   ?= auto

.PHONY: pull-addons harvester-binaries build build-installer package package-harvester package-harvester-webhook package-harvester-upgrade clean $(ENV_FILE)

# ---- Directories ----
$(ROOT)/bin:
	@mkdir -p $@

# ---- Version (regenerate when git state changes) ----
$(ENV_FILE):
	bash $(ROOT)/mk/version-generate $(ROOT)

# ---- Builder image ----
$(MK_DIR)/.builder.stamp: $(ROOT)/mk/Dockerfile.builder
	docker build \
	    --build-arg CONTAINER_WORKDIR=$(CONTAINER_WORKDIR) \
	    --build-arg DAPPER_HOST_ARCH=$(HOST_ARCH) \
	    -f $(ROOT)/mk/Dockerfile.builder \
	    -t $(BUILDER_IMAGE) \
	    $(MK_DIR)
	@touch $@

# ---- Pull addons into local Docker image ----
pull-addons:
	@bash $(MK_DIR)/pull-addons $(MK_DIR) $(ADDONS_IMAGE) $(DOCKER_PROGRESS)

# ---- Compile harvester binaries ----
harvester-binaries: $(MK_DIR)/.builder.stamp $(ENV_FILE) \
    $(shell find $(ROOT)/pkg $(ROOT)/cmd -name '*.go') $(ROOT)/main.go $(ROOT)/go.mod $(ROOT)/go.sum \
    $(MK_DIR)/build-harvester | $(ROOT)/bin
	@bash $(MK_DIR)/docker-build-harvester $(MK_DIR) $(ROOT) $(BIN_IMAGE) $(CONTAINER_WORKDIR) $(DOCKER_PROGRESS)

# ---- Build ----
build: harvester-binaries

# ---- Compile harvester-installer binary ----
$(ROOT)/bin/harvester-installer: $(MK_DIR)/.builder.stamp $(ENV_FILE) pull-addons \
    $(shell find $(ROOT)/installer -name '*.go') $(ROOT)/go.mod $(ROOT)/go.sum \
    $(ROOT)/installer/scripts/build | $(ROOT)/bin
	@bash $(MK_DIR)/docker-build-installer $(MK_DIR) $(ROOT) $(INSTALLER_BIN_IMAGE) $(CONTAINER_WORKDIR) $(DOCKER_PROGRESS)

build-installer: $(ROOT)/bin/harvester-installer

# ---- Package all images ----
package: package-harvester package-harvester-webhook package-harvester-upgrade

# ---- Package harvester image ----
package-harvester: harvester-binaries $(ENV_FILE)
	@bash $(MK_DIR)/package-harvester $(ENV_FILE) $(ROOT) $(DOCKER_PROGRESS)

# ---- Package harvester-webhook image ----
package-harvester-webhook: harvester-binaries $(ENV_FILE)
	@bash $(MK_DIR)/package-harvester-webhook $(ENV_FILE) $(ROOT) $(DOCKER_PROGRESS)

# ---- Package harvester-upgrade image ----
package-harvester-upgrade: harvester-binaries $(ENV_FILE)
	@bash $(MK_DIR)/package-harvester-upgrade $(ENV_FILE) $(ROOT) $(DOCKER_PROGRESS)

# ---- Clean ----
clean:
	rm -rf $(ROOT)/bin
	rm -f $(ROOT)/package/harvester $(ROOT)/package/harvester-webhook $(ROOT)/harvester-env.sh
	rm -f $(ROOT)/package/upgrade/upgrade-helper
	rm -f $(MK_DIR)/.builder.stamp $(MK_DIR)/.addons.stamp

.DEFAULT_GOAL := package-harvester
