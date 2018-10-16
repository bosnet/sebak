## Builder part is quite heavy as it depends on the Go toolchain
## Work around Moby bug 34482
ARG BUILDER
FROM ${BUILDER} AS builder

FROM alpine:latest
LABEL maintainer="BOSCoin Developers <devteam@boscoin.io>"

COPY --from=builder /go/bin/client_test /sebak/
COPY ./entrypoint.sh /sebak/

WORKDIR /sebak/
ENTRYPOINT [ "./entrypoint.sh" ]
CMD []
