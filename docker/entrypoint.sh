#!/bin/sh

if [ -f ".env" ]; then
    source ./.env
fi

# This entrypoint has 2 modes: if any argument is provided to `docker run`, the
# arguments are passed directly to sebak Otherwise, it just starts a node with the
# environment file
if [ $# -gt 0 ]; then
    # Argument mode
    exec ./sebak $@
else
    # Node mode
    exec ./sebak node --genesis=${SEBAK_GENESIS_BLOCK}
fi
