#!/bin/bash

set -xe

# We need to use absolute path for the Docker container
# So make sure we're in the right WD
cd -- `dirname ${BASH_SOURCE[0]}`
TEST_DIRS=$(find . -mindepth 1 -maxdepth 1 -type d -print)
ROOT_DIR=".."
export SEBAK_NODE_ARGS=""
export SEBAK_GENESIS=GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ

# We can only have one trap active at a time, so just save the IDs of containers we started.
# Single quotes around  trap ensure that the variable is evaluated at exit time.
DOCKER_CONTAINERS=""
trap 'if [ ! -z "${DOCKER_CONTAINERS}" ]; then docker rm -f ${DOCKER_CONTAINERS} || true; fi' EXIT

## Build the docker container
NODE_DOCKER_IMAGE=$(docker build -q --build-arg BUILD_MODE='test' --build-arg BUILD_PKG='./cmd/sebak' \
                           --build-arg BUILD_ARGS='-coverpkg=./... -tags integration -c -o /go/bin/sebak' \
                           ${ROOT_DIR} | cut -d: -f2)

# Build the integration test container
TESTS_DOCKER_IMAGE=$(docker build -q . | cut -d: -f2)

if [ -z ${NODE_DOCKER_IMAGE} ] || [ -z ${TESTS_DOCKER_IMAGE} ]; then
    echo "Failed to build at least one docker image" >&2
    exit 1
fi

for dir in ${TEST_DIRS}; do
    # Setup our test environment
    # We need to keep the container around after we stop it when we report coverage,
    # because the reports are written on program's exit, which also means container's shutdown
    # Also SUPER IMPORTANT: the `-test` args need to be before any other args, or they are simply ignored...
    export NODE1=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node1.env \
                          ${NODE_DOCKER_IMAGE} -test.coverprofile=coverage.txt node --genesis=${SEBAK_GENESIS})
    export NODE2=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node2.env \
                          ${NODE_DOCKER_IMAGE} -test.coverprofile=coverage.txt node --genesis=${SEBAK_GENESIS})
    export NODE3=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node3.env \
                          ${NODE_DOCKER_IMAGE} -test.coverprofile=coverage.txt node --genesis=${SEBAK_GENESIS})

    DOCKER_CONTAINERS="${DOCKER_CONTAINERS} ${NODE1} ${NODE2} ${NODE3}"

    # Give them a bit of time
    sleep 10

    # Run the tests
    docker run --rm --network host ${TESTS_DOCKER_IMAGE} ${dir}

    # Shut down the containers - we need to do so for integration reports to be written
    docker stop ${NODE1} ${NODE2} ${NODE3}

    # Copy integration tests
    mkdir -p ${dir}/coverage/node{1,2,3}/
    docker cp ${NODE1}:/sebak/coverage.txt ${dir}/coverage/node1/coverage.txt
    docker cp ${NODE2}:/sebak/coverage.txt ${dir}/coverage/node2/coverage.txt
    docker cp ${NODE3}:/sebak/coverage.txt ${dir}/coverage/node3/coverage.txt

    # Cleanup
    docker rm -f ${NODE1} ${NODE2} ${NODE3} || true
done
