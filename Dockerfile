FROM golang:1.24-trixie AS build-backend

WORKDIR /build

# Copy go module files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies with mount cache
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy only necessary source directories
# NOTE: internal/static/build should be pre-built and included in the build context
COPY cmd/ cmd/
COPY internal/ internal/

# Build with CGO enabled (required for Solace library)
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /go/bin/solar-controller \
    ./cmd/controller

FROM debian:trixie-slim

COPY --from=build-backend /go/bin/solar-controller /

ENV GIN_MODE=release

CMD ["/solar-controller"]