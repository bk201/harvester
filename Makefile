ROOT              := $(realpath $(dir $(realpath $(firstword $(MAKEFILE_LIST)))))
MK_DIR            := $(ROOT)/mk
BOLD              := \033[1m
CYAN              := \033[36m
GREEN             := \033[32m
YELLOW            := \033[33m
RESET             := \033[0m
BUILDER_IMAGE          := harvester-builder:local
BIN_IMAGE              := harvester-bin:local
INSTALLER_BIN_IMAGE    := harvester-installer-bin:local
ADDONS_IMAGE           := harvester-addons:local
BUNDLE_BUILDER_IMAGE   := harvester-bundle-builder:local
BUNDLE_IMAGE           := harvester-bundle:local
CONTAINER_WORKDIR := /go/src/github.com/harvester/harvester
ENV_FILE          := $(ROOT)/harvester-env.sh
HOST_ARCH         := $(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
DOCKER_PROGRESS   ?= plain
export DOCKER_PROGRESS

.PHONY: builder-image pull-addons harvester-binaries build build-installer bundle-builder-image build-bundle package package-harvester package-harvester-webhook package-harvester-upgrade ci clean

# ---- Directories ----
$(ROOT)/bin:
	@mkdir -p $@

# ---- Version (regenerate when git state changes) ----
$(ENV_FILE):
	@printf "$(BOLD)$(CYAN)===> Generating version metadata$(RESET)\n"
	@bash $(MK_DIR)/version/generate $(ROOT)

# ---- Builder image ----
builder-image:
	@printf "$(BOLD)$(CYAN)===> Building builder image$(RESET)\n"
	@docker build \
	    --build-arg CONTAINER_WORKDIR=$(CONTAINER_WORKDIR) \
	    --build-arg DAPPER_HOST_ARCH=$(HOST_ARCH) \
	    -f $(MK_DIR)/Dockerfile.builder.build \
	    -t $(BUILDER_IMAGE) \
	    $(MK_DIR)

# ---- Pull addons into local Docker image ----
pull-addons: builder-image
	@printf "$(BOLD)$(CYAN)===> Pulling addons$(RESET)\n"
	@bash $(MK_DIR)/$@/docker-build $(ADDONS_IMAGE)

# ---- Compile harvester binaries ----
harvester-binaries: builder-image $(ENV_FILE) | $(ROOT)/bin
	@printf "$(BOLD)$(CYAN)===> Building harvester binaries (harvester, harvester-webhook, upgrade-helper)$(RESET)\n"
	@bash $(MK_DIR)/$@/docker-build $(BIN_IMAGE) $(CONTAINER_WORKDIR)

# ---- Build ----
build: harvester-binaries


# ---- Validate ----
validate: builder-image
	@printf "$(BOLD)$(CYAN)===> Validating harvester GO sources$(RESET)\n"
	@bash $(MK_DIR)/$@/docker-build

# ---- Test ----
test: builder-image
	@printf "$(BOLD)$(CYAN)===> Running harvester unit tests$(RESET)\n"
	@bash $(MK_DIR)/$@/docker-build

# ---- Test integration ----
test-integration: builder-image $(ENV_FILE)
	@printf "$(BOLD)$(CYAN)===> Running harvester integration tests$(RESET)\n"
	@bash $(MK_DIR)/$@/docker-build $(CONTAINER_WORKDIR)

# ---- Compile harvester-installer binary ----
build-installer: builder-image $(ENV_FILE) pull-addons | $(ROOT)/bin
	@printf "$(BOLD)$(CYAN)===> Building harvester-installer binary$(RESET)\n"
	@bash $(MK_DIR)/$@/docker-build $(INSTALLER_BIN_IMAGE) $(CONTAINER_WORKDIR)

# ---- Bundle builder image (extends harvester-builder with addons) ----
bundle-builder-image: builder-image pull-addons
	@printf "$(BOLD)$(CYAN)===> Building bundle builder image$(RESET)\n"
	@docker build \
	    --progress=$(DOCKER_PROGRESS) \
	    -f $(MK_DIR)/Dockerfile.builder.bundle \
	    -t $(BUNDLE_BUILDER_IMAGE) \
	    $(MK_DIR)

# ---- Build offline bundle (charts + images) ----
build-bundle: bundle-builder-image package $(ENV_FILE)
	@printf "$(BOLD)$(CYAN)===> Building offline bundle (requires network access)$(RESET)\n"
	@bash $(MK_DIR)/$@/docker-build $(BUNDLE_IMAGE) $(ENV_FILE) $(CONTAINER_WORKDIR)

# ---- Package all images ----
package: package-harvester package-harvester-webhook package-harvester-upgrade

# ---- Package harvester image ----
package-harvester: harvester-binaries $(ENV_FILE)
	@printf "$(BOLD)$(GREEN)===> Packaging harvester image$(RESET)\n"
	@bash $(MK_DIR)/package-harvester/package $(ENV_FILE) $(ROOT)

# ---- Package harvester-webhook image ----
package-harvester-webhook: harvester-binaries $(ENV_FILE)
	@printf "$(BOLD)$(GREEN)===> Packaging harvester-webhook image$(RESET)\n"
	@bash $(MK_DIR)/package-harvester-webhook/package $(ENV_FILE) $(ROOT)

# ---- Package harvester-upgrade image ----
package-harvester-upgrade: harvester-binaries build-installer $(ENV_FILE)
	@printf "$(BOLD)$(GREEN)===> Packaging harvester-upgrade image$(RESET)\n"
	@bash $(MK_DIR)/package-harvester-upgrade/package $(ENV_FILE) $(ROOT) $(BIN_IMAGE) $(INSTALLER_BIN_IMAGE) $(CONTAINER_WORKDIR)

# ---- Clean ----
clean:
	@printf "$(BOLD)$(YELLOW)===> Cleaning build artifacts$(RESET)\n"
	@rm -rf $(ROOT)/bin
	@rm -f $(ROOT)/package/harvester $(ROOT)/package/harvester-webhook $(ROOT)/harvester-env.sh
	@rm -f $(MK_DIR)/.addons.stamp


very-clean: clean
	@printf "$(BOLD)$(YELLOW)===> Removing builder images images$(RESET)\n"
	@docker rmi -f $(BUILDER_IMAGE) $(BIN_IMAGE) $(INSTALLER_BIN_IMAGE) $(ADDONS_IMAGE) $(BUNDLE_BUILDER_IMAGE) $(BUNDLE_IMAGE) || true

.DEFAULT_GOAL := package-harvester


ci: validate build test
