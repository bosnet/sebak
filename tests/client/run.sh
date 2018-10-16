#!/bin/bash

set -xe
cd -- `dirname ${BASH_SOURCE[0]}`

ROOT_DIR="../.."

export SEBAK_GENESIS=GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ
export SEBAK_COMMON=GDYIHSHMDXJ4MXE35N4IMNC2X3Q3F665C5EX2JWHHCUW2PCFVXIFEE2C

NODE=""
function dumpLogsAndCleanup () {
    if [ ! -z "${NODE}" ]; then
        docker logs ${NODE} || true
        docker rm -f ${NODE} || true
    fi
}

trap dumpLogsAndCleanup EXIT

## Build the node runner docker image
NODE_IMAGE=$(docker build -q \
    --build-arg BUILD_MODE="test" \
    --build-arg BUILD_PKG="./cmd/sebak" \
    --build-arg BUILD_ARGS="-coverpkg=./... -tags integration -c -o /go/bin/sebak"  \
    ${ROOT_DIR}/ | cut -d: -f2)

if [ -z ${NODE_IMAGE} ]; then
    echo "Failed to build node runner docker image" >&2
    exit 1
fi

## Build the docker builder image
IMAGE=$(docker build -q \
    --build-arg BUILD_MODE="test" \
    --build-arg BUILD_PKG="./tests/client" \
    --build-arg BUILD_ARGS="-c -o /go/bin/sdk_test"  \
    ${ROOT_DIR}/ -f ${ROOT_DIR}/Dockerfile_client.build | cut -d: -f2)

if [ -z ${IMAGE} ]; then
    echo "Failed to build builder docker image" >&2
    exit 1
fi

TEST_IMAGE=$(docker build -q --build-arg BUILDER=${IMAGE} \
    . | cut -d: -f2)

if [ -z ${TEST_IMAGE} ]; then
    echo "Failed to build tester docker image" >&2
    exit 1
fi


NODE=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/self.env \
      ${NODE_IMAGE} node \
      --genesis=${SEBAK_GENESIS},${SEBAK_COMMON} \
      --log-level=debug)

sleep 1

docker run --rm --network host ${TEST_IMAGE} $@
# Shut down the containers - we need to do so for integration reports to be written
docker stop ${NODE}
# Cleanup
docker rm -f ${NODE} || true
