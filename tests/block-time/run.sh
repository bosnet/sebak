#!/bin/bash

set -xe

# We need to use absolute path for the Docker container
# So make sure we're in the right WD
cd -- `dirname ${BASH_SOURCE[0]}`
ROOT_DIR="../.."
export SEBAK_GENESIS=GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ
export SEBAK_COMMON=GDYIHSHMDXJ4MXE35N4IMNC2X3Q3F665C5EX2JWHHCUW2PCFVXIFEE2C

CONTAINERS=""
function dumpLogsAndCleanup () {
    if [ ! -z "${CONTAINERS}" ]; then
        for CONTAINER in ${CONTAINERS}; do
            docker logs ${CONTAINER} || true
        done
        docker rm -f ${CONTAINERS} || true
    fi
}

trap dumpLogsAndCleanup EXIT

## Build the docker container
NODE_IMAGE=$(docker build -q --build-arg BUILD_MODE='test' --build-arg BUILD_PKG='./cmd/sebak' \
                           --build-arg BUILD_ARGS='-coverpkg=./... -tags integration -c -o /go/bin/sebak' \
                           ${ROOT_DIR} | cut -d: -f2)

CLIENT_IMAGE=$(docker build -q . | cut -d: -f2)

if [ -z ${NODE_IMAGE} ] || [ -z ${CLIENT_IMAGE} ]; then
    echo "Failed to build at least one docker image" >&2
    exit 1
fi

# Setup our test environment
# We need to keep the container around after we stop it when we report coverage,
# because the reports are written on program's exit, which also means container's shutdown
# Also SUPER IMPORTANT: the `-test` args need to be before any other args, or they are simply ignored...
export NODE1=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node1.env \
                        ${NODE_IMAGE} node --log-level=debug \
                        --genesis=${SEBAK_GENESIS},${SEBAK_COMMON})
export NODE2=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node2.env \
                        ${NODE_IMAGE} node --log-level=debug \
                        --genesis=${SEBAK_GENESIS},${SEBAK_COMMON})
export NODE3=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node3.env \
                        ${NODE_IMAGE} node --log-level=debug \
                        --genesis=${SEBAK_GENESIS},${SEBAK_COMMON})
export NODE4=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node4.env \
                        ${NODE_IMAGE} node --log-level=debug \
                        --genesis=${SEBAK_GENESIS},${SEBAK_COMMON})

CONTAINERS="${CONTAINERS} ${NODE1} ${NODE2} ${NODE3} ${NODE4}"

SECONDS=0

# Give that a bit of time
sleep 60

docker run --rm --network host ${CLIENT_IMAGE} block-time.sh ${SECONDS}

# Shut down the containers - we need to do so for integration reports to be written
docker stop ${NODE1} ${NODE2} ${NODE3} ${NODE4}

# Cleanup
docker rm -f ${NODE1} ${NODE2} ${NODE3} ${NODE4} || true
