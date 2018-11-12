#!/bin/bash

set -xe


if [ $# -ne 1 ] && [ $# -ne 2 ]; then
    echo 1>&2 "Error: Expected 1(test directory) or 2 argument (block_time.sh SECONDS), $# provided..."
    exit 1
fi

if [ $# -eq 1 ]; then
  if [ -d "${1}" ]
  then
    echo "run test"
    TEST_DIR=${1}
    ./default_runner.sh ${TEST_DIR}
  else
    ./${1}
  fi
fi

if [ $# -eq 2 ]; then
    echo "check block time"
   ./${1} ${2}
fi

