## Builder part is quite heavy as it depends on the Go toolchain
FROM alpine:latest
LABEL maintainer="BOSCoin Developers <devteam@boscoin.io>"

COPY --from=sebak:builder_sdk /go/bin/sdk /sebak/
COPY ./entrypoint.sh /sebak/

WORKDIR /sebak/
ENTRYPOINT [ "./entrypoint.sh" ]
CMD []
