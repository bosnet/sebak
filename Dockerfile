## This one is much more lightweight
FROM alpine:latest AS runner

RUN apk --no-cache add ca-certificates      # For SSL requests
COPY docker/entrypoint.sh /sebak/entrypoint.sh
COPY docker/sebak.* /sebak/
COPY --from=sebak:builder /go/bin/sebak /sebak/

# Make it so that nodes have the same ID and genesis block by default
ENV SEBAK_NETWORK_ID    sebak-test-network

WORKDIR /sebak/
ENTRYPOINT [ "./entrypoint.sh" ]
CMD []
