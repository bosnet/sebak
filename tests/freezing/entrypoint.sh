#!/bin/bash

set -uxe
set -o pipefail

source utils.sh

## The test is as follows:
## - Create a frozen account, record height (h0), wait 10s
## - Do a payment operation, wait 60s
## - Ensure nothing happens and height < (h0 + 20)
## - Wait another 60s, ensure that height >= (h0 + 20) and that the balances match

SEBAK_GENESIS=GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ
FROZEN_ACCOUNT=GDTEPFWEITKFHSUO44NQABY2XHRBBH2UBVGJ2ZJPDREIOL2F6RAEBJE4

GENESIS_BALANCE="9999999899999990000"
FROZEN_ACCOUNT_BALANCE="100000000000"
GENESIS_FINAL__="9999999999999980000"

HEIGHT=0

curl --insecure \
     --request POST \
     --header "Content-Type: application/json" \
     --data "$(cat 1_create_account.json)" \
     https://127.0.0.1:2821/node/message \
     >/dev/null 2>&1

sleep 10
for ((port=2821;port<=2823;port++)); do
    height=$(getBlockHeight ${port})
    if [ "$height" -ge "6" ]; then
        die "Expected height to be under 6, not ${height}"
    fi
    genesisBalance=$(getAccountWithBalance ${port} ${SEBAK_GENESIS} ${GENESIS_BALANCE})
    if [ $? -ne 0 ];then
        die "Expected genesis balance to be ${GENESIS_BALANCE}, not ${genesisBalance}"
    fi
    frozenBalance=$(getAccountWithBalance ${port} ${FROZEN_ACCOUNT} ${FROZEN_ACCOUNT_BALANCE})
    if [ $? -ne 0 ];then
        die "Expected genesis balance to be ${FROZEN_ACCOUNT_BALANCE}, not ${frozenBalance}"
    fi
    # 1 extra block for safety
    HEIGHT=$(($height+21))
done

curl --insecure \
     --request POST \
     --header "Content-Type: application/json" \
     --data "$(cat 2_unfreeze.json)" \
     https://127.0.0.1:2822/node/message \
     >/dev/null 2>&1

sleep 60
for ((port=2821;port<=2823;port++)); do
    height=$(getBlockHeight ${port})
    while [ "$height" -ge "${HEIGHT}" ]; do
        sleep 5
        height=$(getBlockHeight ${port})
    done
    # When an account is unfreezed, its balance goes to 0 immediately,
    # but the receiving account is not credited
    genesisBalance=$(getAccountWithBalance ${port} ${SEBAK_GENESIS} ${GENESIS_BALANCE})
    if [ $? -ne 0 ];then
        die "Expected genesis balance to be ${GENESIS_BALANCE}, not ${genesisBalance}"
    fi
    frozenBalance=$(getAccountWithBalance ${port} ${FROZEN_ACCOUNT} "0")
    if [ $? -ne 0 ];then
        die "Expected genesis balance to be 0, not ${frozenBalance}"
    fi
done

sleep 60
for ((port=2821;port<=2823;port++)); do
    height=$(getBlockHeight ${port})
    while [ "$height" -le "${HEIGHT}" ]; do
        sleep 5
        height=$(getBlockHeight ${port})
    done
    frozenBalance=$(getAccountWithBalance ${port} ${FROZEN_ACCOUNT} "0")
    if [ $? -ne 0 ];then
        die "Expected genesis balance to be 0, not ${frozenBalance}"
    fi
    genesisBalance=$(getAccountWithBalance ${port} ${SEBAK_GENESIS} ${GENESIS_FINAL__})
    if [ $? -ne 0 ];then
        die "Expected genesis balance to be ${GENESIS_FINAL__}, not ${genesisBalance}"
    fi
done
