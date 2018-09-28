#!/bin/bash

set -xe
cd -- `dirname ${BASH_SOURCE[0]}`
ROOT_DIR="../.."
NODE_NAME="SEBAKNODE"
IMAGE_TAG="sebak:test"

docker build -q -t ${IMAGE_TAG} \
    --build-arg BUILD_MODE='test' \
    --build-arg BUILD_PKG='./cmd/sebak' \
    --build-arg BUILD_ARGS='-tags integration -c -o /go/bin/sebak' \
    ${ROOT_DIR}

docker rmi $(docker images --filter dangling=true -q --no-trunc) || true
docker images