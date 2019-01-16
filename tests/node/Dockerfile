## Work around Moby bug 34482
ARG BUILDER
FROM ${BUILDER} AS builder

## We do not have any dependency on Sebak itself and should not
FROM alpine:latest

RUN apk --no-cache add bash curl jq

# Some test utility
COPY --from=builder /sebak-tools/accstreamer /usr/sbin/
## Copy the full directory
ADD . /tests/

WORKDIR /tests/
ENTRYPOINT [ "/tests/entrypoint.sh" ]
CMD []
