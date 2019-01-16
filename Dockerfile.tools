## Build the tools
FROM golang:1.11-alpine AS builder
LABEL maintainer="BOSCoin Developers <devteam@boscoin.io>"

RUN apk add --no-cache git gcc musl-dev

COPY ./ /sebak-tools/
WORKDIR /sebak-tools/

RUN go build -o accstreamer ./tools/accstreamer/main.go
