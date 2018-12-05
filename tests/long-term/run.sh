#!/bin/bash

set -xe

# We need to use absolute path for the Docker container
# So make sure we're in the right WD
cd -- `dirname ${BASH_SOURCE[0]}`
ROOT_DIR="../.."
export BLOCK_TIME_DELTA=2s
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
                        ${NODE_IMAGE} node --log-level=debug --rate-limit-api=0-s --rate-limit-node=0-s \
                        --genesis=${SEBAK_GENESIS},${SEBAK_COMMON},5000000000000000 --block-time-delta=${BLOCK_TIME_DELTA})
export NODE2=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node2.env \
                        ${NODE_IMAGE} node --log-level=debug --rate-limit-api=0-s --rate-limit-node=0-s \
                        --genesis=${SEBAK_GENESIS},${SEBAK_COMMON},5000000000000000 --block-time-delta=${BLOCK_TIME_DELTA})
export NODE3=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node3.env \
                        ${NODE_IMAGE} node --log-level=debug --rate-limit-api=0-s --rate-limit-node=0-s \
                        --genesis=${SEBAK_GENESIS},${SEBAK_COMMON},5000000000000000 --block-time-delta=${BLOCK_TIME_DELTA})
export NODE4=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node4.env \
                        ${NODE_IMAGE} node --log-level=debug --rate-limit-api=0-s --rate-limit-node=0-s \
                        --genesis=${SEBAK_GENESIS},${SEBAK_COMMON},5000000000000000 --block-time-delta=${BLOCK_TIME_DELTA})

CONTAINERS="${CONTAINERS} ${NODE1} ${NODE2} ${NODE3} ${NODE4}"

SECONDS=0

# Give that a bit of time
sleep 1

docker run --rm --network host ${CLIENT_IMAGE} "create_account"
docker run --rm --network host ${CLIENT_IMAGE} "payment_1"

COUNT=2
for i in $(seq $COUNT)
do
    # Rerun NODE4 for checking sync
    docker stop ${NODE4}
    docker rm -f ${NODE4}

    export NODE4=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node4.env \
                            ${NODE_IMAGE} -test.coverprofile=coverage.txt \
                            node --log-level=debug --rate-limit-api=0-s --rate-limit-node=0-s \
                            --genesis=${SEBAK_GENESIS},${SEBAK_COMMON},5000000000000000 --block-time-delta=${BLOCK_TIME_DELTA})

    CONTAINERS="${CONTAINERS} ${NODE4}"

    # Check that the block height of all 4 nodes are almost same and all CONSENSUS state
    docker run --rm --network host ${CLIENT_IMAGE} sync.sh

    docker stop ${NODE3}
    docker rm -f ${NODE3}

    export NODE3=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node3.env \
                            ${NODE_IMAGE} -test.coverprofile=coverage.txt \
                            node --log-level=debug --rate-limit-api=0-s --rate-limit-node=0-s \
                            --genesis=${SEBAK_GENESIS},${SEBAK_COMMON},5000000000000000 --block-time-delta=${BLOCK_TIME_DELTA})

    CONTAINERS="${CONTAINERS} ${NODE3}"

    docker run --rm --network host ${CLIENT_IMAGE} sync.sh

    docker stop ${NODE2}
    docker rm -f ${NODE2}

    export NODE2=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node2.env \
                            ${NODE_IMAGE} -test.coverprofile=coverage.txt \
                            node --log-level=debug --rate-limit-api=0-s --rate-limit-node=0-s \
                            --genesis=${SEBAK_GENESIS},${SEBAK_COMMON},5000000000000000 --block-time-delta=${BLOCK_TIME_DELTA})

    CONTAINERS="${CONTAINERS} ${NODE2}"

    docker run --rm --network host ${CLIENT_IMAGE} sync.sh

    docker stop ${NODE1}
    docker rm -f ${NODE1}

    export NODE1=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node1.env \
                            ${NODE_IMAGE} -test.coverprofile=coverage.txt \
                            node --log-level=debug --rate-limit-api=0-s --rate-limit-node=0-s \
                            --genesis=${SEBAK_GENESIS},${SEBAK_COMMON},5000000000000000 --block-time-delta=${BLOCK_TIME_DELTA})

    CONTAINERS="${CONTAINERS} ${NODE1}"

    docker run --rm --network host ${CLIENT_IMAGE} sync.sh

    docker run --rm --network host ${CLIENT_IMAGE} "total_balance"

done

docker run --rm --network host ${CLIENT_IMAGE} "payment_2"
docker run --rm --network host ${CLIENT_IMAGE} "common_account"
docker run --rm --network host ${CLIENT_IMAGE} "total_balance"

# Shut down the containers - we need to do so for integration reports to be written
docker stop ${NODE1} ${NODE2} ${NODE3} ${NODE4}

# Copy integration tests
mkdir -p ./long-term/coverage/node{1,2,3,4}/
docker cp ${NODE1}:/sebak/coverage.txt ./long-term/coverage/node1/coverage.txt
docker cp ${NODE2}:/sebak/coverage.txt ./long-term/coverage/node2/coverage.txt
docker cp ${NODE3}:/sebak/coverage.txt ./long-term/coverage/node3/coverage.txt
docker cp ${NODE4}:/sebak/coverage.txt ./long-term/coverage/node4/coverage.txt

# Cleanup
docker rm -f ${NODE1} ${NODE2} ${NODE3} ${NODE4} || true
