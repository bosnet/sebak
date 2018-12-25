#!/bin/bash

set -xe

source utils.sh

HEIGHT_BEFORE=$(getBlockHeight 2821)

sleep 10

HEIGHT_AFTER=$(getBlockHeight 2821)

if [ "${HEIGHT_BEFORE}" == "${HEIGHT_AFTER}" ]; then
    die "Ten seconds later, expected height was not ${HEIGHT_BEFORE}, but it was ${HEIGHT_AFTER}"
fi
