#!/bin/bash

set -xe

# We need to use absolute path for the Docker container
# So make sure we're in the right WD
cd -- `dirname ${BASH_SOURCE[0]}`
ROOT_DIR=".."

# Build the docker container
${ROOT_DIR}/build.sh "test" "./cmd/sebak" "-coverpkg=./... -tags integration -c -o /go/bin/sebak"

./api/run.sh
./sdk/run.sh
