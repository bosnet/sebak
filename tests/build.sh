#!/bin/bash

set -xe

# We need to use absolute path for the Docker container
# So make sure we're in the right WD
cd -- `dirname ${BASH_SOURCE[0]}`

ROOT_DIR=".."

if [ $# -lt 2 ]; then
    echo "the number of parameter less than 2" >&2
    echo "build.sh build_mode builde_pkg [build_args]" >&2
    exit 1
fi

BUILD_MODE="install"
BUILD_PKG="./..."
BUILD_ARGS=""

if [ -n $1 ]; then
    BUILD_MODE=$1
fi

if [ -n $2 ]; then
    BUILD_PKG=$2
fi

if [ -n "$3" ]; then
    BUILD_ARGS=$3
fi

## Build the docker image
IMAGE=$(docker build --tag sebak:runner -q \
    --build-arg BUILD_MODE=${BUILD_MODE} \
    --build-arg BUILD_PKG=${BUILD_PKG} \
    --build-arg BUILD_ARGS="${BUILD_ARGS}"  \
    ${ROOT_DIR}/ | cut -d: -f2)

if [ -z ${IMAGE} ]; then
    echo "Failed to build builder docker image" >&2
    exit 1
fi
