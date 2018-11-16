#!/bin/bash

set -xe

source utils.sh

# Give that a bit of time for syncing
sleep 60

# check height from nodes
HEIGHT1=$(getBlockHeight 2821)
HEIGHT2=$(getBlockHeight 2822)
HEIGHT3=$(getBlockHeight 2823)
HEIGHT4=$(getBlockHeight 2824)

EXPECTED1="$HEIGHT1"
EXPECTED2=`expr $HEIGHT1 + 1`
EXPECTED3=`expr $HEIGHT1 - 1`

if [ "${EXPECTED1}" != "${HEIGHT2}" ] && [ "$EXPECTED2" != "${HEIGHT2}" ] && [ "$EXPECTED3" != "${HEIGHT2}" ] ; then
    die "Expected height of the node4 to be $HEIGHT1, not ${HEIGHT2} by consensus"
fi

if [ "${EXPECTED1}" != "${HEIGHT3}" ] && [ "$EXPECTED2" != "${HEIGHT3}" ] && [ "$EXPECTED3" != "${HEIGHT3}" ] ; then
    die "Expected height of the node4 to be $HEIGHT1, not ${HEIGHT3} by consensus"
fi

if [ "${EXPECTED1}" != "${HEIGHT4}" ] && [ "$EXPECTED2" != "${HEIGHT4}" ] && [ "$EXPECTED3" != "${HEIGHT4}" ] ; then
    die "Expected height of the node4 to be $HEIGHT1, not ${HEIGHT4} by sync"
fi

STATE=$(getNodeState 2821)

if [ "${STATE}" != "\"CONSENSUS\"" ]; then
    die "Expected state of the node1 to be CONSENSUS, not ${STATE}"
fi

STATE=$(getNodeState 2822)

if [ "${STATE}" != "\"CONSENSUS\"" ]; then
    die "Expected state of the node2 to be CONSENSUS, not ${STATE}"
fi

STATE=$(getNodeState 2823)

if [ "${STATE}" != "\"CONSENSUS\"" ]; then
    die "Expected state of the node3 to be CONSENSUS, not ${STATE}"
fi

STATE=$(getNodeState 2824)

if [ "${STATE}" != "\"CONSENSUS\"" ]; then
    die "Expected state of the node4 to be CONSENSUS, not ${STATE}"
fi
