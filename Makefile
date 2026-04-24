ROOT              := $(realpath $(dir $(realpath $(firstword $(MAKEFILE_LIST)))))
MK_DIR            := $(ROOT)/mk
BOLD              := \033[1m
CYAN              := \033[36m
GREEN             := \033[32m
YELLOW            := \033[33m
RESET             := \033[0m

# User might have several repos in a host. Distinguish each by using the abs path of the repo
MK_REPO_ID                := $(shell echo -n "$(ROOT)$$(cat /etc/machine-id 2>/dev/null)" | sha256sum | cut -c1-8)

MK_BUILDER_IMAGE         := harvester-builder:$(MK_REPO_ID)
MK_BIN_IMAGE             := harvester-bin:$(MK_REPO_ID)
MK_INSTALLER_BIN_IMAGE   := harvester-installer-bin:$(MK_REPO_ID)
MK_ADDONS_IMAGE          := harvester-addons:$(MK_REPO_ID)
MK_BUNDLE_BUILDER_IMAGE  := harvester-bundle-builder:$(MK_REPO_ID)
MK_BUNDLE_IMAGE          := harvester-bundle:$(MK_REPO_ID)
MK_CONTAINER_WORKDIR     := /go/src/github.com/harvester/harvester
MK_ENV_FILE              := $(ROOT)/mk-env.sh
MK_ENV_FILE_NAME         := $(notdir $(MK_ENV_FILE))
MK_DOCKER_PROGRESS              ?= auto
MK_DOCKER_BUILD_NO_CACHE        ?= false
MK_ROOT                         := $(ROOT)
MK_HARVESTER_INSTALLER_REPO     ?= http://192.168.2.22:3000/kiefer/harvester-installer.git
MK_HARVESTER_INSTALLER_REF      ?= wip-buildx
MK_RKE2_IMAGE_REPO              ?=
MK_USE_LOCAL_IMAGES             ?=
MK_REPO                         ?= rancher

export MK_DOCKER_PROGRESS MK_CONTAINER_WORKDIR MK_BUILDER_IMAGE MK_BIN_IMAGE MK_INSTALLER_BIN_IMAGE MK_REPO_ID
export MK_ADDONS_IMAGE MK_BUNDLE_BUILDER_IMAGE MK_BUNDLE_IMAGE MK_ENV_FILE MK_ENV_FILE_NAME MK_DOCKER_BUILD_NO_CACHE MK_ROOT
export MK_HARVESTER_INSTALLER_REPO MK_HARVESTER_INSTALLER_REF MK_RKE2_IMAGE_REPO MK_USE_LOCAL_IMAGES MK_ISO_REPO

HOST_ARCH              := $(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

.PHONY: env $(MK_ENV_FILE) builder-image pull-addons harvester-binaries build build-installer bundle-builder-image \
	build-bundle build-iso package package-harvester package-harvester-webhook package-harvester-upgrade ci \
	arm clean default generate-addons validate-ci generate-manifest generate-openapi

# ---- Directories ----
$(ROOT)/bin:
	@mkdir -p $@

# ---- Version (regenerate when git state changes) ----
$(MK_ENV_FILE):
	@printf "$(BOLD)$(CYAN)===> Generating version metadata$(RESET)\n"
	@bash $(MK_DIR)/version/generate $(MK_ENV_FILE)

env: $(MK_ENV_FILE)
	
# ---- Builder image ----
builder-image:
	@printf "$(BOLD)$(CYAN)===> Building builder image$(RESET)\n"
	@docker build \
	    --build-arg CONTAINER_WORKDIR=$(MK_CONTAINER_WORKDIR) \
	    --build-arg DAPPER_HOST_ARCH=$(HOST_ARCH) \
	    -f $(MK_DIR)/Dockerfile.builder.build \
	    -t $(MK_BUILDER_IMAGE) \
	    $(MK_DIR)

# ---- Pull addons into local Docker image ----
pull-addons: builder-image
	@printf "$(BOLD)$(CYAN)===> Pulling addons$(RESET)\n"
	@bash $(MK_DIR)/$@/docker-build

# ---- Compile harvester binaries ----
harvester-binaries: builder-image $(MK_ENV_FILE) | $(ROOT)/bin
	@printf "$(BOLD)$(CYAN)===> Building harvester binaries (harvester, harvester-webhook, upgrade-helper)$(RESET)\n"
	@bash $(MK_DIR)/$@/docker-build

# ---- Build ----
build: harvester-binaries


# ---- Validate ----
validate: builder-image
	@printf "$(BOLD)$(CYAN)===> Validating harvester GO sources$(RESET)\n"
	@bash $(MK_DIR)/$@/docker-build

# ---- Validate CI (dirty check after go generate + go mod tidy) ----
validate-ci: builder-image
	@printf "$(BOLD)$(CYAN)===> Validating CI environment (dirty check)$(RESET)\n"
	@bash $(MK_DIR)/$@/docker-build

# ---- Test ----
test: builder-image
	@printf "$(BOLD)$(CYAN)===> Running harvester unit tests$(RESET)\n"
	@bash $(MK_DIR)/$@/docker-build

# ---- Test integration ----
test-integration: builder-image $(MK_ENV_FILE)
	@printf "$(BOLD)$(CYAN)===> Running harvester integration tests$(RESET)\n"
	@bash $(MK_DIR)/$@/docker-build

# ---- Compile harvester-installer binary ----
build-installer: builder-image $(MK_ENV_FILE) pull-addons | $(ROOT)/bin
	@printf "$(BOLD)$(CYAN)===> Building harvester-installer binary$(RESET)\n"
	@bash $(MK_DIR)/$@/docker-build

# ---- Bundle builder image (extends harvester-builder with addons) ----
bundle-builder-image: builder-image pull-addons
	@printf "$(BOLD)$(CYAN)===> Building bundle builder image$(RESET)\n"
	@docker build \
	    --progress=$(MK_DOCKER_PROGRESS) \
	    -f $(MK_DIR)/Dockerfile.builder.bundle \
	    -t $(MK_BUNDLE_BUILDER_IMAGE) \
	    $(MK_DIR)

# ---- Build ISO ----
build-iso: builder-image $(MK_ENV_FILE)
	@printf "$(BOLD)$(CYAN)===> Building Harvester ISO$(RESET)\n"
	@bash $(MK_DIR)/$@/docker-build

# ---- Build offline bundle (charts + images) ----
build-bundle: bundle-builder-image package $(MK_ENV_FILE)
	@printf "$(BOLD)$(CYAN)===> Building offline bundle (requires network access)$(RESET)\n"
	@bash $(MK_DIR)/$@/docker-build

# ---- Package harvester image ----
package-harvester: harvester-binaries $(MK_ENV_FILE)
	@printf "$(BOLD)$(GREEN)===> Packaging harvester image$(RESET)\n"
	@bash $(MK_DIR)/$@/package

# ---- Package harvester-webhook image ----
package-harvester-webhook: harvester-binaries $(MK_ENV_FILE)
	@printf "$(BOLD)$(GREEN)===> Packaging harvester-webhook image$(RESET)\n"
	@bash $(MK_DIR)/$@/package

# ---- Package harvester-upgrade image ----
package-harvester-upgrade: harvester-binaries build-installer $(MK_ENV_FILE)
	@printf "$(BOLD)$(GREEN)===> Packaging harvester-upgrade image$(RESET)\n"
	@bash $(MK_DIR)/$@/package

# ---- Package all images ----
package: package-harvester package-harvester-webhook package-harvester-upgrade
	
# ---- Generate CRD manifests ----
generate-manifest: builder-image
	@printf "$(BOLD)$(CYAN)===> Generating CRD manifests$(RESET)\n"
	@bash $(MK_DIR)/$@/docker-build

# ---- Generate OpenAPI/Swagger spec ----
generate-openapi: builder-image
	@printf "$(BOLD)$(CYAN)===> Generating OpenAPI/Swagger spec$(RESET)\n"
	@bash $(MK_DIR)/$@/docker-build

# ---- Generate Add-ons manifests ----
generate-addons:
	@printf "$(BOLD)$(CYAN)===> Building add-ons manifests$(RESET)\n"
	@bash $(MK_DIR)/$@/build
	
# ---- Clean ----
clean:
	@printf "$(BOLD)$(YELLOW)===> Cleaning build artifacts$(RESET)\n"
	@rm -rf $(ROOT)/bin
	@rm -f $(ROOT)/package/harvester $(ROOT)/package/harvester-webhook $(MK_ENV_FILE)
	@rm -f $(ROOT)/package/upgrade/upgrade-helper $(ROOT)/package/upgrade/harvester-installer
	@rm -rf $(ROOT)/package/upgrade/addons
	@rm -rf $(ROOT)/dist/pull-addons
	@rm -rf $(ROOT)/dist/artifacts $(ROOT)/dist/harvester-cluster-repo
	@rm -f $(MK_DIR)/.addons.stamp


clean-all: clean
	@printf "$(BOLD)$(YELLOW)===> Removing builder images images$(RESET)\n"
	@docker rmi -f $(MK_BUILDER_IMAGE) $(MK_BIN_IMAGE) $(MK_INSTALLER_BIN_IMAGE) $(MK_ADDONS_IMAGE) $(MK_BUNDLE_BUILDER_IMAGE) $(MK_BUNDLE_IMAGE) || true
	@docker rmi -f harvester-generate-manifest:$(MK_REPO_ID) harvester-generate-openapi:$(MK_REPO_ID) harvester-iso-builder:$(MK_REPO_ID) || true

.DEFAULT_GOAL := default

default: build test package

arm: build package

ci: validate validate-ci build test package-harvester-webhook package-harvester-upgrade \
	test-integration package-harvester
