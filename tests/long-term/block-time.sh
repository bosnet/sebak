#!/bin/bash

set -xe

source utils.sh

# After 60s, block height will be near 12
sleep 60
# check height from nodes
for ((port=2821;port<=2823;port++)); do
    height=$(getBlockHeight ${port})
    if [ "$height" != "11" ] && [ "$height" != "12" ] && [ "$height" != "13" ] ; then
        die "Expected height to be 11, 12 or 13, not ${height}"
    fi
done

# After 120s from the start, block height will be near 24
sleep 60
# check height from nodes
for ((port=2821;port<=2823;port++)); do
    height=$(getBlockHeight ${port})
    if [ "$height" != "23" ] && [ "$height" != "24" ] && [ "$height" != "25" ] ; then
        die "Expected height to be 23, 24 or 25, not ${height}"
    fi
done
