#!/bin/bash

set -xe
cd -- `dirname ${BASH_SOURCE[0]}`

TEST_DIRS=$(find . -mindepth 1 -maxdepth 1 -type d -print)
ROOT_DIR="../.."

export SEBAK_GENESIS=GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ
export SEBAK_COMMON=GDYIHSHMDXJ4MXE35N4IMNC2X3Q3F665C5EX2JWHHCUW2PCFVXIFEE2C

source ./build.sh

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

for dir in ${TEST_DIRS}; do
    # Setup our test environment
    # We need to keep the container around after we stop it when we report coverage,
    # because the reports are written on program's exit, which also means container's shutdown
    # Also SUPER IMPORTANT: the `-test` args need to be before any other args, or they are simply ignored...

    for index in 1 2 3; do
        NODE=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/node${index}.env \
                       sebak:runner \
                       -test.coverprofile=coverage.txt \
                       node \
                       --genesis=${SEBAK_GENESIS},${SEBAK_COMMON} \
                       --log-level=debug)
        DOCKER_CONTAINERS="${DOCKER_CONTAINERS} ${NODE}"
    done

    # Give them a bit of time
    sleep 1

    # Run the tests
    docker run --rm --network host sebak:api_tester ${dir}

        # Copy integration tests
    mkdir -p ${dir}/coverage/node{1,2,3}/
    index=1
    for CONTAINER in ${DOCKER_CONTAINERS}; do
        docker cp ${CONTAINER}:/sebak/coverage.txt ${dir}/coverage/node${index}/coverage.txt
        index = ${index} + 1
    done

    # Shut down the containers - we need to do so for integration reports to be written
    docker stop ${DOCKER_CONTAINERS}
    # Cleanup
    docker rm -f ${DOCKER_CONTAINERS} || true
done