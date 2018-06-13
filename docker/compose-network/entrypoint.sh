#!/bin/sh

set -e
set -x

env | sort

cd /go/src/github.com/owlchain/sebak

if [ ! -z ${SEBAK_SRC_COMMAND} ];then
    ${SEBAK_SRC_COMMAND}
fi

cd /go/src/github.com/owlchain/sebak/cmd/sebak

go run main.go genesis ${SEBAK_GENESIS_BLOCK}

for VALIDATOR in ${SEBAK_VALIDATORS}; do
    VALIDATOR_ARGS="${VALIDATOR_ARGS} --validator=${VALIDATOR}"
done

go run main.go node \
    --network-id=${SEBAK_NETWORK_ID} \
    --secret-seed=${SEBAK_SECRET_SEED} \
    ${VALIDATOR_ARGS} \
    $*
