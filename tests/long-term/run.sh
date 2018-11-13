#!/bin/bash

set -xe

# We need to use absolute path for the Docker container
# So make sure we're in the right WD
cd -- `dirname ${BASH_SOURCE[0]}`
ROOT_DIR="../.."
export SEBAK_NODE_ARGS=""
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
                        ${NODE_IMAGE} node --genesis=${SEBAK_GENESIS},${SEBAK_COMMON} --log-level=debug)
export NODE2=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node2.env \
                        ${NODE_IMAGE} node --genesis=${SEBAK_GENESIS},${SEBAK_COMMON} --log-level=debug)
export NODE3=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node3.env \
                        ${NODE_IMAGE} node --genesis=${SEBAK_GENESIS},${SEBAK_COMMON} --log-level=debug)
export NODE4=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node4.env \
                        ${NODE_IMAGE} node --genesis=${SEBAK_GENESIS},${SEBAK_COMMON} --log-level=debug)

CONTAINERS="${CONTAINERS} ${NODE1} ${NODE2} ${NODE3} ${NODE4}"

SECONDS=0

# Give that a bit of time
sleep 1

docker run --rm --network host ${CLIENT_IMAGE} "create_account"
docker run --rm --network host ${CLIENT_IMAGE} "payment_1"
docker run --rm --network host ${CLIENT_IMAGE} "total_balance"

# Check block height after 60s
docker run --rm --network host ${CLIENT_IMAGE} block-time.sh ${SECONDS}

# Rerun NODE4 for checking sync
docker stop ${NODE4}
docker rm -f ${NODE4}

export NODE4_2=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node4.env \
                        ${NODE_IMAGE} -test.coverprofile=coverage.txt node --genesis=${SEBAK_GENESIS},${SEBAK_COMMON} --log-level=debug)

CONTAINERS="${CONTAINERS} ${NODE4_2}"

# Check that the block height of all 4 nodes are almost same
docker run --rm --network host ${CLIENT_IMAGE} sync.sh

# Rerun NODE3 for checking sync
docker stop ${NODE3}
docker rm -f ${NODE3}

export NODE3_2=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node3.env \
                        ${NODE_IMAGE} -test.coverprofile=coverage.txt node --genesis=${SEBAK_GENESIS},${SEBAK_COMMON} --log-level=debug)

CONTAINERS="${CONTAINERS} ${NODE3_2}"

# Check that the block height of all 4 nodes are almost same
docker run --rm --network host ${CLIENT_IMAGE} sync.sh

# Rerun NODE2 for checking sync
docker stop ${NODE2}
docker rm -f ${NODE2}

export NODE2_2=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node2.env \
                        ${NODE_IMAGE} -test.coverprofile=coverage.txt node --genesis=${SEBAK_GENESIS},${SEBAK_COMMON} --log-level=debug)

CONTAINERS="${CONTAINERS} ${NODE2_2}"

# Check that the block height of all 4 nodes are almost same
docker run --rm --network host ${CLIENT_IMAGE} sync.sh

# Rerun NODE1 for checking sync
docker stop ${NODE1}
docker rm -f ${NODE1}

export NODE1_2=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node1.env \
                        ${NODE_IMAGE} -test.coverprofile=coverage.txt node --genesis=${SEBAK_GENESIS},${SEBAK_COMMON} --log-level=debug)

CONTAINERS="${CONTAINERS} ${NODE1_2}"

# Check that the block height of all 4 nodes are almost same
docker run --rm --network host ${CLIENT_IMAGE} sync.sh

# Check again after sync
docker run --rm --network host ${CLIENT_IMAGE} "payment_2"
docker run --rm --network host ${CLIENT_IMAGE} "common_account"
docker run --rm --network host ${CLIENT_IMAGE} "total_balance"

# Shut down the containers - we need to do so for integration reports to be written
docker stop ${NODE1_2} ${NODE2_2} ${NODE3_2} ${NODE4_2}

# Copy integration tests
mkdir -p ./long-term/coverage/node{1,2,3,4}/
docker cp ${NODE1_2}:/sebak/coverage.txt ./long-term/coverage/node1/coverage.txt
docker cp ${NODE2_2}:/sebak/coverage.txt ./long-term/coverage/node2/coverage.txt
docker cp ${NODE3_2}:/sebak/coverage.txt ./long-term/coverage/node3/coverage.txt
docker cp ${NODE4_2}:/sebak/coverage.txt ./long-term/coverage/node4/coverage.txt

# Cleanup
docker rm -f ${NODE1_2} ${NODE2_2} ${NODE3_2} ${NODE4_2} || true
