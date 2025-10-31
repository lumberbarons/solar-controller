FROM node:22-alpine AS build-frontend

WORKDIR /app/site

# Copy package files first for better layer caching
COPY site/package.json site/package-lock.json ./

# Use npm ci for faster, more reliable installs and mount cache for npm packages
RUN --mount=type=cache,target=/root/.npm \
    npm ci --only=production --no-audit

# Copy source files and build
COPY site/ ./

RUN npm run build

FROM golang:1.24-alpine AS build-backend

WORKDIR /build

# Copy go module files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies with mount cache
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy only necessary source directories
COPY cmd/ cmd/
COPY internal/ internal/

# Copy frontend build artifacts from previous stage
COPY --from=build-frontend /app/site/build internal/static/build

# Build with optimizations for smaller binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /go/bin/solar-controller \
    ./cmd/controller

FROM debian:trixie-slim

COPY --from=build-backend /go/bin/solar-controller /

ENV GIN_MODE=release

CMD ["/solar-controller"]