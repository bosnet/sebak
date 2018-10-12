#!/bin/bash

cd -- `dirname ${BASH_SOURCE[0]}`

ROOT_DIR="../.."

BUILD_MODE="install"
BUILD_PKG="./..."
BUILD_ARGS=""

## Build the docker builder image
IMAGE=$(docker build --tag sebak:builder -q \
    --build-arg BUILD_MODE=${BUILD_MODE} \
    --build-arg BUILD_PKG=${BUILD_PKG} \
    --build-arg BUILD_ARGS="${BUILD_ARGS}"  \
    ${ROOT_DIR}/ -f ${ROOT_DIR}/Dockerfile.build | cut -d: -f2)

if [ -z ${IMAGE} ]; then
    echo "Failed to build builder docker image" >&2
    exit 1
fi

IMAGE=$(docker build --tag sebak:sdk_tester -q \
    . | cut -d: -f2)

if [ -z ${IMAGE} ]; then
    echo "Failed to build tester docker image" >&2
    exit 1
fi
