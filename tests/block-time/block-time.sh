#!/bin/bash

set -xe

source utils.sh

if [ $# -ne 1 ]; then
    die "Expected 1 argument (SECONDS), not $#: $@"
fi

SECONDS=${1}
DIV=`expr $SECONDS / 5`
EXPECTED1=`expr $DIV`
EXPECTED2=`expr $EXPECTED1 - 1`
EXPECTED3=`expr $EXPECTED1 + 1`

# check height from nodes
for ((port=2821;port<=2824;port++)); do
    HEIGHT=$(getBlockHeight ${port})
    if [ "$HEIGHT" != "$EXPECTED1" ] && [ "$HEIGHT" != "$EXPECTED2" ] && [ "$HEIGHT" != "$EXPECTED3" ] ; then
        die "Expected height to be around $EXPECTED1, not ${HEIGHT}"
    fi
done
