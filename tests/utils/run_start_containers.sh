#!/bin/bash
set -xe
cd -- `dirname ${BASH_SOURCE[0]}`
ROOT_DIR="../.."
NODE_NAME="SEBAKNODE"
IMAGE_TAG="sebak:test"
SEBAK_GENESIS=GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ

for INDEX in 1 2 3
do

docker run -d --network host --name ${NODE_NAME}-${INDEX} --env-file=${ROOT_DIR}/docker/node${INDEX}.env \
       $IMAGE_TAG node --genesis=${SEBAK_GENESIS}\
       --log-level=debug --timeout-init=4 --timeout-sign=4 --timeout-accept=4 --block-time=10
done

docker container ls