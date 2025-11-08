FROM debian:trixie-slim

# Copy pre-built binary from build context
COPY bin/solar-controller /

ENV GIN_MODE=release

CMD ["/solar-controller"]