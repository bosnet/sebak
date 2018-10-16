#!/bin/bash

set -xe
# We need to use absolute path for the Docker container
# So make sure we're in the right WD
cd -- `dirname ${BASH_SOURCE[0]}`

TEST_DIRS=$(find . -mindepth 1 -maxdepth 1 -type d -print)
ROOT_DIR="../.."

export SEBAK_GENESIS=GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ
export SEBAK_COMMON=GDYIHSHMDXJ4MXE35N4IMNC2X3Q3F665C5EX2JWHHCUW2PCFVXIFEE2C

# We can only have one trap active at a time, so just save the IDs of containers we started.
# Single quotes around  trap ensure that the variable is evaluated at exit time.
DOCKER_CONTAINERS=""
function dumpLogsAndCleanup () {
    if [ ! -z "${DOCKER_CONTAINERS}" ]; then
        for CONTAINER in ${DOCKER_CONTAINERS}; do
            docker logs ${CONTAINER} || true
        done
        docker rm -f ${DOCKER_CONTAINERS} || true
    fi
}

trap dumpLogsAndCleanup EXIT

## Build the node runner docker image
IMAGE=$(docker build --tag sebak:runner -q \
    --build-arg BUILD_MODE="test" \
    --build-arg BUILD_PKG="./cmd/sebak" \
    --build-arg BUILD_ARGS="-coverpkg=./... -tags integration -c -o /go/bin/sebak"  \
    ${ROOT_DIR}/ | cut -d: -f2)

if [ -z ${IMAGE} ]; then
    echo "Failed to build node runner docker image" >&2
    exit 1
fi


# Build the integration test docker image
IMAGE=$(docker build --tag sebak:api_tester -q \
    . | cut -d: -f2)

if [ -z ${IMAGE} ]; then
    echo "Failed to build tester docker image" >&2
    exit 1
fi

for dir in ${TEST_DIRS}; do
    # Setup our test environment
    # We need to keep the container around after we stop it when we report coverage,
    # because the reports are written on program's exit, which also means container's shutdown
    # Also SUPER IMPORTANT: the `-test` args need to be before any other args, or they are simply ignored...
    NODES=""
    for index in 1 2 3; do
        NODE=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node${index}.env \
                       sebak:runner \
                       -test.coverprofile=coverage.txt \
                       node \
                       --genesis=${SEBAK_GENESIS},${SEBAK_COMMON} \
                       --log-level=debug)
        DOCKER_CONTAINERS="${DOCKER_CONTAINERS} ${NODE}"
        NODES="${NODES} ${NODE}"
    done

    # Give them a bit of time
    sleep 1

    # Run the tests
    docker run --rm --network host sebak:api_tester ${dir}

    # Shut down the containers - we need to do so for integration reports to be written
    docker stop ${NODES}

    # Copy integration tests
    mkdir -p ${dir}/coverage/node{1,2,3}/
    index=1
    for NODE in ${NODES}; do
        docker cp ${NODE}:/sebak/coverage.txt ${dir}/coverage/node${index}/coverage.txt
        index=`expr ${index} + 1`
    done

    # Cleanup
    docker rm -f ${NODES} || true
done