#!/bin/sh

set -xe

if [ $# -eq 1 ]; then
    exec ./sdk_test -test.run $@
else
    exec ./sdk_test
fi

