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
    starttime=$(date +%s)

    PORT=$(echo "${JSONFILE}" | cut -d'_' -f3)
    curl --insecure \
         --request POST \
         --header "Content-Type: application/json" \
         --data "$(cat ${JSONFILE})" \
         https://127.0.0.1:${PORT}/api/v1/transactions \
         >/dev/null 2>&1

    echo ${JSONFILE} "- Elapsed Time:" $(expr $(date +%s) - ${starttime}) "seconds"
done

# Intermediate checks
if [ -f ./${DIR}.check ]; then
    bash -c ./${DIR}.check
fi
