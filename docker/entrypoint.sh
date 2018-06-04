#!/bin/sh

## Allow us to split the $SEBAK_VALIDATORS 'array' into multiple `--validator` args
## Note: It's not a proper array since the shell we use is `sh`, not `bash`
./sebak genesis ${SEBAK_GENESIS_BLOCK}
for VALIDATOR in ${SEBAK_VALIDATORS}; do
    VALIDATOR_ARGS="${VALIDATOR_ARGS} --validator=${VALIDATOR}"
done
./sebak node ${VALIDATOR_ARGS} --network-id=${SEBAK_NETWORK_ID} --secret-seed=${SEBAK_SECRET_SEED} --log-level debug
