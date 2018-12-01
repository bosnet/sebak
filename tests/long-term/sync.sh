#!/bin/bash

set -xe

source utils.sh

sleep 20

for i in $(seq 50)
do
    ALL_DONE=1
    for ((port=2821;port<=2824;port++))
    do
        STATE=$(getNodeState ${port})
        if [ $STATE != "\"CONSENSUS\"" ]; then
            ALL_DONE=0
            break
        fi
    done
    if [ $ALL_DONE == 1 ]
    then
        break
    else
        sleep 1
    fi
done

if [ $ALL_DONE == 0 ]; then
    die "no sync for 1 minute"
fi

# check height from nodes
HEIGHT1=$(getBlockHeight 2821)
HEIGHT2=$(getBlockHeight 2822)
HEIGHT3=$(getBlockHeight 2823)
HEIGHT4=$(getBlockHeight 2824)

EXPECTED1="$HEIGHT1"
EXPECTED2=`expr $HEIGHT1 + 1`
EXPECTED3=`expr $HEIGHT1 - 1`

if [ "${EXPECTED1}" != "${HEIGHT2}" ] && [ "$EXPECTED2" != "${HEIGHT2}" ] && [ "$EXPECTED3" != "${HEIGHT2}" ] ; then
    die "Expected height of the node4 to be $HEIGHT1, not ${HEIGHT2}"
fi

if [ "${EXPECTED1}" != "${HEIGHT3}" ] && [ "$EXPECTED2" != "${HEIGHT3}" ] && [ "$EXPECTED3" != "${HEIGHT3}" ] ; then
    die "Expected height of the node4 to be $HEIGHT1, not ${HEIGHT3}"
fi

if [ "${EXPECTED1}" != "${HEIGHT4}" ] && [ "$EXPECTED2" != "${HEIGHT4}" ] && [ "$EXPECTED3" != "${HEIGHT4}" ] ; then
    die "Expected height of the node4 to be $HEIGHT1, not ${HEIGHT4}"
fi
