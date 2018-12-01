#!/bin/bash

set -xe

if [ $# -ne 2 ] ; then
    echo 1>&2 "Error: Expected 2 args block_time.sh $(SECONDS), $# provided..."
    exit 1
fi


if [ $# -eq 2 ]; then
   ./${1} ${2}
fi
