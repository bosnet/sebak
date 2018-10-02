#!/bin/bash

IMAGE=$(docker build --tag sebak:sdk_tester -q \
    . | cut -d: -f2)

if [ -z ${IMAGE} ]; then
    echo "Failed to build tester docker image" >&2
    exit 1
fi
