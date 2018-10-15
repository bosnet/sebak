## Builder part is quite heavy as it depends on the Go toolchain
FROM golang:1.11-alpine AS builder
LABEL maintainer="BOSCoin Developers <devteam@boscoin.io>"


RUN apk add --no-cache git openssh gcc musl-dev linux-headers
RUN go get github.com/ahmetb/govvv

#golang:alpine set $GOPATH to `/go`
COPY ./ /sebak-build
WORKDIR /sebak-build

# You probably don't need to change this
ARG BUILD_MODE="install"
ARG BUILD_ARGS=''
ARG BUILD_PKG="./..."

RUN go $BUILD_MODE $BUILD_ARGS -ldflags="$(govvv -pkg boscoin.io/sebak/lib/version -flags)" -v $BUILD_PKG

## This one is much more lightweight
FROM alpine:latest AS runner

RUN apk --no-cache add ca-certificates      # For SSL requests
COPY docker/entrypoint.sh /sebak/entrypoint.sh
COPY docker/sebak.* /sebak/
COPY --from=builder /go/bin/sebak /sebak/

# Make it so that nodes have the same ID and genesis block by default
ENV SEBAK_NETWORK_ID    sebak-test-network

WORKDIR /sebak/
ENTRYPOINT [ "./entrypoint.sh" ]
CMD []
