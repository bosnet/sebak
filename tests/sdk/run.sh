#!/bin/bash

set -xe
cd -- `dirname ${BASH_SOURCE[0]}`

ROOT_DIR="../.."

./build.sh

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

NODE=$(docker run -d --network host --env-file=${ROOT_DIR}/docker/self.env \
      sebak:runner node \
      --genesis=${SEBAK_GENESIS},${SEBAK_COMMON} \
      --log-level=debug)

sleep 1

docker run --rm --network host sebak:sdk_tester
# Shut down the containers - we need to do so for integration reports to be written
docker stop ${NODE}
# Cleanup
docker rm -f ${NODE} || true
