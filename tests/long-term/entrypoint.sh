#!/bin/bash

set -xe

if [ $# -ne 1 ]; then
    echo 1>&2 "Error: Expected 1(test directory or executable), $# provided..."
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
