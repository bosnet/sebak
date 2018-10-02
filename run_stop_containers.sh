#!/bin/bash
set -xe

NODE_NAME="SEBAKNODE"

docker stop ${NODE_NAME}-1 ${NODE_NAME}-2 ${NODE_NAME}-3
docker rm ${NODE_NAME}-1 ${NODE_NAME}-2 ${NODE_NAME}-3
