#!/bin/bash

set -xe

# We need to use absolute path for the Docker container
# So make sure we're in the right WD
cd -- `dirname ${BASH_SOURCE[0]}`
ROOT_DIR=".."

# Build the docker container
source ${ROOT_DIR}/build.sh "install" "./..."

cd -- `dirname ${BASH_SOURCE[0]}`

./api/run.sh
./sdk/run.sh
