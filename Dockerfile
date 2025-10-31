FROM golang:1.24-alpine AS build-backend

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

# Build with optimizations for smaller binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /go/bin/solar-controller \
    ./cmd/controller

FROM debian:trixie-slim

COPY --from=build-backend /go/bin/solar-controller /

ENV GIN_MODE=release

CMD ["/solar-controller"]