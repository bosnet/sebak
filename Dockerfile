## Builder part is quite heavy as it depends on the Go toolchain
FROM golang:alpine AS builder
LABEL maintainer="BOSCoin Developers <devteam@boscoin.io>"

#golang:alpine set $GOPATH to `/go`
COPY ./ /go/src/boscoin.io/sebak
WORKDIR /go/src/boscoin.io/sebak

## Since those need to be re-run every time anyway, they are in a single stage
# `git` and `openssh` are needed for `go get`
RUN apk add --no-cache git openssh          \
    && go get github.com/golang/dep/cmd/dep \
    && dep ensure -v                        \
    && go install -v ./...

## This one is much more lightweight
FROM alpine:latest AS runner

RUN apk --no-cache add ca-certificates      # For SSL requests
COPY docker/entrypoint.sh /sebak/entrypoint.sh
COPY docker/sebak.* /sebak/
COPY --from=builder /go/bin/sebak /sebak/

# Make it so that nodes have the same ID and genesis block by default
ENV SEBAK_NETWORK_ID    sebak-test-network

WORKDIR /sebak/
ENTRYPOINT ./entrypoint.sh
