#!/bin/sh

set -xe

if [ $# -eq 1 ]; then
    exec ./client_test -test.run $@
else
    exec ./client_test
fi

