## Builder part is quite heavy as it depends on the Go toolchain
FROM golang:alpine AS builder
LABEL maintainer="BOSCoin Developers <devteam@boscoin.io>"

# You probably don't need to change this
ARG BUILD_MODE="install"
ARG BUILD_ARGS=''
ARG BUILD_PKG="./..."

RUN apk add --no-cache git openssh gcc musl-dev linux-headers
RUN go get github.com/ahmetb/govvv

#golang:alpine set $GOPATH to `/go`
COPY ./ /go/src/boscoin.io/sebak
WORKDIR /go/src/boscoin.io/sebak

## Note that we do not get the dependencies anew
## We carry over whatever is in `vendor`, so the user MUST run `dep ensure` in their local copy
## This make building the container orders of magnitude faster (`dep ensure` is extremely slow),
## greatly reduce the container's size, and gives more control to the user as to what is tested
## (one can replace a dependency, if needed).
## Otherwise, install dep and dependencies.(e.g Docker Hub automated build)

RUN if [ ! -d vendor ]; then go get github.com/golang/dep/cmd/dep; dep ensure -v; fi

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
