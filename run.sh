#!/bin/bash
set -xe

if [ $# -lt 2 ]; then
    echo "run.sh {env_file_location} {db_(keep/new)}"
fi

ENV_LOCATION=$1

source ${ENV_LOCATION}
SEBAK_STORAGE="/tmp/sebak_test/db"

if [ $2 == "db_new" ]; then
    rm -rf ${SEBAK_STORAGE}
    sebak node \
    --network-id sebak \
    --validators ${SEBAK_VALIDATORS} \
    --secret-seed ${SEBAK_SECRET_SEED} \
    --tls-cert ./docker/sebak.crt \
    --tls-key ./docker/sebak.key \
    --bind ${SEBAK_BIND} \
    --storage "file://"${SEBAK_STORAGE} \
    --genesis=${SEBAK_GENESIS_BLOCK},${SEBAK_COMMON_ACCOUNT}
fi

if [ $2 == "db_keep" ]; then
    rm -rf ${SEBAK_STORAGE}
    sebak node \
    --network-id sebak \
    --validators ${SEBAK_VALIDATORS} \
    --secret-seed ${SEBAK_SECRET_SEED} \
    --tls-cert ./docker/sebak.crt \
    --tls-key ./docker/sebak.key \
    --bind ${SEBAK_BIND} \
    --storage "file://"${SEBAK_STORAGE}
fi
