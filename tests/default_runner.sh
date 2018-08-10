#!/bin/bash

set -xe
source utils.sh

if [ $# -ne 1 ]; then
    die "Expected 1 argument (directory), not $#: $@"
fi

DIR=${1}

cd $DIR
echo "===== ${DIR}: Default test runner ====="

for JSONFILE in $(find . -name "*.json" -type f | sort); do
    PORT=$(echo "${JSONFILE}" | cut -d'_' -f3)
    curl --insecure \
         --request POST \
         --header "Content-Type: application/json" \
         --data "$(cat ${JSONFILE})" \
         https://127.0.0.1:${PORT}/node/message \
         >/dev/null 2>&1
    # Intermediate checks
    if [ -f ${JSONFILE}.check ]; then
        bash -c ${JSONFILE}.check
    fi
done
