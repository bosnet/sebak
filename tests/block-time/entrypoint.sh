#!/bin/bash

set -xe

function die () # string message
{
    MESSAGE=${1:-"An error happened, but no message was provided"}
    echo 1>&2 "Error: ${MESSAGE}"
    exit 1
}

function getBlockHeight ()
{
    if [ $# -ne 1 ]; then
        die "1 arguments expected for getBlockHeight, not $#"
    fi

    echo $(curl --insecure \
         --request GET \
         --header "Accept: application/json" \
         https://127.0.0.1:$1/ \
         2>/dev/null \
         | jq ".block.height" )
}

# After 60s, block height will be 11, 12 or 13
sleep 60
# check height from nodes
for ((port=2821;port<=2823;port++)); do
    height=$(getBlockHeight ${port})
    if [ "$height" != "11" ] && [ "$height" != "12" ] && [ "$height" != "13" ] ; then
        die "Expected height to be 11, 12 or 13, not ${height}"
    fi
done

# After 120s from the start, block height will be 23, 24 or 25
sleep 60
# check height from nodes
for ((port=2821;port<=2823;port++)); do
    height=$(getBlockHeight ${port})
    if [ "$height" != "23" ] && [ "$height" != "24" ] && [ "$height" != "25" ] ; then
        die "Expected height to be 23, 24 or 25, not ${height}"
    fi
done

