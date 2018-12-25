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

function getNodeState ()
{
    if [ $# -ne 1 ]; then
        die "1 arguments expected for getNodeState, not $#"
    fi

    echo $(curl --insecure \
         --request GET \
         --header "Accept: application/json" \
         https://127.0.0.1:$1/ \
         2>/dev/null \
         | jq ".node.state" )
}
