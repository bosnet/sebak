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

function getAccount () # u16 port, string addr
{
    if [ $# -ne 2 ]; then
        die "2 arguments expected for getAccount, not $#"
    fi

    echo $(curl --insecure \
                  --request GET \
                  --header "Accept: application/json" \
                  https://127.0.0.1:${1}/api/v1/accounts/${2} \
                  2>/dev/null)
}

function getAccountWithBalance () # u16 port, string addr and expected balance
{
    if [ $# -ne 3 ]; then
        die "3 arguments expected for getAccountWithBalance, not $#"
    fi

    balance=0
    for i in $(seq 30)
    do
	    balance=$(getAccount $1 $2 | jq ".balance" | sed 's/"//g')
	    if [ "$balance" == "$3" ];then
		    return 0
	    fi
	    sleep 0.3
    done

    echo $balance

    return 1
}
