#!/bin/bash
# Shared library functions for mk/ scripts

# Extract a file from a Docker image
extract_binary() {
    local image="$1"
    local container_path="$2"
    local host_path="$3"
    local container_name="$4"

    echo "Extracting file from image..."
    echo "  Image: $image:/$container_path -> host:/$host_path"

    # Clean up any stale container from a previous failed run
    docker rm -f "$container_name" >/dev/null 2>&1 || true
    trap "docker rm -f '$container_name' >/dev/null 2>&1 || true" EXIT

    docker create --name "$container_name" "$image" >/dev/null
    docker cp "$container_name:$container_path" "$host_path"
    docker rm "$container_name" >/dev/null

    trap - EXIT
}

# Set MK_DOCKER_BUILD_ARGS to the standard docker build flags.
# Usage: set_mk_docker_build_args; docker build "${MK_DOCKER_BUILD_ARGS[@]}" ...
set_mk_docker_build_args() {
    MK_DOCKER_BUILD_ARGS=(
        --progress="${MK_DOCKER_PROGRESS}"
        --build-arg MK_REPO_ID
        --build-arg MK_BUILDER_IMAGE
        --build-arg MK_CONTAINER_WORKDIR
        --build-arg MK_ENV_FILE_NAME
    )
}

# Extract a directory from a Docker image
# extract /abc to /path/to/save, will copy abc directory to /path/to/save/
extract_directory() {
    local image="$1"
    local container_path="$2"
    local host_path="$3"
    local container_name="$4"

    echo "Extracting directory from image..."
    echo "  Image: $image:/$container_path -> host:/$host_path"

    # Clean up any stale container from a previous failed run
    docker rm -f "$container_name" >/dev/null 2>&1 || true
    trap "docker rm -f '$container_name' >/dev/null 2>&1 || true" EXIT

    # mkdir -p ensures docker cp always sees an existing directory and places
    # the source dir inside it, giving consistent $host_path/<basename> output
    # regardless of whether the destination already existed.
    mkdir -p "$host_path"
    docker create --name "$container_name" "$image" >/dev/null
    docker cp "$container_name:$container_path" "$host_path"
    docker rm "$container_name" >/dev/null

    trap - EXIT
}
