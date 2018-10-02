#!/bin/bash

set -xe

if [ $# -ne 1 ]; then
    echo 1>&2 "Error: Expected 1 argument (test directory), $# provided..."
    exit 1
fi

# Make sure we're in the right directory
cd -- `dirname ${BASH_SOURCE[0]}`
TEST_DIR=${1}

## Test suites can override the runner should they need to
if [ -x ${TEST_DIR}/run.sh ] && [ -f ${TEST_DIR}/run.sh ]; then
    exit 1
else
    ./default_runner.sh ${TEST_DIR}
fi
