FROM debian:trixie-slim

# Copy pre-built binary from build context
COPY bin/solar-controller /

ENV GIN_MODE=release

# Run as a non-root user; mounted configs must be readable by this user
RUN groupadd --system --gid 65532 solar \
    && useradd --system --uid 65532 --gid solar --no-create-home solar
USER solar

CMD ["/solar-controller"]
