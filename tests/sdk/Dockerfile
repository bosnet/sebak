## Builder part is quite heavy as it depends on the Go toolchain
ARG BUILDER
FROM ${BUILDER} AS builder

FROM alpine:latest
LABEL maintainer="BOSCoin Developers <devteam@boscoin.io>"

COPY --from=builder /go/bin/sdk /sebak/
COPY ./entrypoint.sh /sebak/

WORKDIR /sebak/
ENTRYPOINT [ "./entrypoint.sh" ]
CMD []
