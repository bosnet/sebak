#!/bin/bash

set -xe

source utils.sh

if [ $# -ne 1 ]; then
    die "Expected 1 argument (SECONDS), not $#: $@"
fi

SECONDS=${1}

# After 60s from the start, block height will be near 12
sleep $((60 - $SECONDS))
# check height from nodes
for ((port=2821;port<=2823;port++)); do
    HEIGHT=$(getBlockHeight ${port})
    if [ "$HEIGHT" != "11" ] && [ "$HEIGHT" != "12" ] && [ "$HEIGHT" != "13" ] ; then
        die "Expected height to be 11, 12 or 13, not ${HEIGHT}"
    fi
done
