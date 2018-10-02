#!/bin/bash

set -xe

# We need to use absolute path for the Docker container
# So make sure we're in the right WD
cd -- `dirname ${BASH_SOURCE[0]}`

if [ $# -lt 2 ]; then
    echo "the number of parameter less than 2" >&2
    echo "build.sh build_mode builde_pkg [build_args]" >&2
    exit 1
fi

BUILD_MODE="install"
BUILD_PKG="./..."
BUILD_ARGS=""
if [ $# -eq 3 ]; then
    BUILD_ARGS=$3
fi

## Build the docker builder image
IMAGE=$(docker build --tag sebak:builder -q \
    --build-arg BUILD_MODE=${BUILD_MODE} \
    --build-arg BUILD_PKG=${BUILD_PKG} \
    --build-arg BUILD_ARGS=${BUILD_ARGS}  \
    . -f ./Dockerfile.build | cut -d: -f2)

if [ -z ${IMAGE} ]; then
    echo "Failed to build builder docker image" >&2
    exit 1
fi

## Build the docker runner image
IMAGE=$(docker build --tag sebak:runner -q \
    . | cut -d: -f2)

if [ -z ${IMAGE} ]; then
    echo "Failed to build runner docker image" >&2
    exit 1
fi
