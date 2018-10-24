#!/bin/bash

set -xe

if [ $# -ne 1 ]; then
    echo 1>&2 "Error: Expected 1 argument (test directory), $# provided..."
    exit 1
fi

./$1
