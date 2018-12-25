#!/bin/bash

set -xe

# We need to use absolute path for the Docker container
# So make sure we're in the right WD
cd -- `dirname ${BASH_SOURCE[0]}`
ROOT_DIR="../.."
CURRENT_DIR=$(echo "${PWD##*/}")

source accounts.sh
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
                        ${NODE_IMAGE} -test.coverprofile=coverage.txt \
                        node --log-level=debug --rate-limit-api=0-s --rate-limit-node=0-s \
                        --genesis=${SEBAK_GENESIS},${SEBAK_COMMON},5000000000000000)
export NODE2=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node2.env \
                        ${NODE_IMAGE} -test.coverprofile=coverage.txt \
                        node --log-level=debug --rate-limit-api=0-s --rate-limit-node=0-s \
                        --genesis=${SEBAK_GENESIS},${SEBAK_COMMON},5000000000000000)
export NODE3=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node3.env \
                        ${NODE_IMAGE} -test.coverprofile=coverage.txt \
                        node --log-level=debug --rate-limit-api=0-s --rate-limit-node=0-s \
                        --genesis=${SEBAK_GENESIS},${SEBAK_COMMON},5000000000000000)

CONTAINERS="${CONTAINERS} ${NODE1} ${NODE2} ${NODE3}"

SECONDS=0

# Give that a bit of time
sleep 1

# Check sync
docker run --rm --network host ${CLIENT_IMAGE} sync.sh

docker stop ${NODE3}
sleep 10
docker start ${NODE3}

# Check sync after starting NODE3
docker run --rm --network host ${CLIENT_IMAGE} consensus_alive.sh

# Shut down the containers - we need to do so for integration reports to be written
docker stop ${NODE1} ${NODE2} ${NODE3}

# Copy integration tests
mkdir -p ./${CURRENT_DIR}/coverage/node{1,2,3}/
docker cp ${NODE1}:/sebak/coverage.txt ./${CURRENT_DIR}/coverage/node1/coverage.txt
docker cp ${NODE2}:/sebak/coverage.txt ./${CURRENT_DIR}/coverage/node2/coverage.txt
docker cp ${NODE3}:/sebak/coverage.txt ./${CURRENT_DIR}/coverage/node3/coverage.txt

# Cleanup
docker rm -f ${NODE1} ${NODE2} ${NODE3} || true
