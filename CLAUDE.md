# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Solar-controller is a Go-based service that collects metrics from solar power equipment (Epever) and publishes them via MQTT and Prometheus. It includes a React-based web UI for monitoring.

## Development Commands

### Using Make (Recommended)

```bash
# Show available make targets
make help

# Build everything (frontend + backend)
make build

# Build only frontend (React app)
make build-frontend

# Build only backend (Go binary)
make build-backend

# Build backend for Linux ARM64
make build-linux-arm64

# Run tests
make test

# Clean build artifacts
make clean
```

### Backend (Go)

```bash
# Build the application (requires frontend to be built first)
go build -o bin/solar-controller ./cmd/controller

# Run with configuration
./bin/solar-controller -config path/to/config.yaml

# Run in debug mode
./bin/solar-controller -config path/to/config.yaml -debug

# Run tests
go test ./...

# Install dependencies
go get -d -v ./...

# Tidy dependencies
go mod tidy
```

### Frontend (React)

```bash
cd site

# Install dependencies
npm install

# Run development server (proxies to backend on port 8000)
npm start

# Build for production
npm run build

# Run tests
npm test
```

### Docker

```bash
# Build Docker image
docker build -t solar-controller .

# Run container
docker run solar-controller -config /etc/solar-controller/config.yaml
```

### Release

Uses goreleaser for multi-platform builds and packaging:

```bash
# Create release (requires git tag)
goreleaser release

# Test release build without publishing
goreleaser release --snapshot --clean
```

## Architecture

### Controller Pattern

The application uses a plugin-style controller architecture where each hardware type implements the `SolarController` interface:

```go
type SolarController interface {
    RegisterEndpoints(r *gin.Engine)
    Enabled() bool
}
```

The Epever controller follows this structure:
- **Controller**: Main orchestrator that manages scheduled collection/publishing
- **Collector**: Handles device communication and metric collection
- **Configurer**: Manages device configuration
- **PrometheusCollector**: Exposes metrics to Prometheus

Controllers are instantiated in `main.go:buildControllers()` and conditionally enabled based on configuration. Each controller has an `enabled` boolean field that must be set to `true` for the controller to start. If required config fields are missing (even when enabled is true), the controller returns an empty/disabled instance.

### Data Flow

1. Controllers are initialized with YAML configuration at startup
2. Each enabled controller schedules periodic collection via `gocron`
3. On each tick, the controller's `collectAndPublish()` method:
   - Calls the Collector to fetch device metrics
   - Updates Prometheus metrics via PrometheusCollector
   - Publishes JSON payload to message broker (MQTT or Solace) via MessagePublisher interface
   - Caches last status for HTTP API endpoints

### Message Publishing

The application supports two message broker options (mutually exclusive):

- **MQTT**: Using Eclipse Paho MQTT client
  - QoS 0 (fire-and-forget)
  - 5-second publish timeout
  - Suitable for lightweight deployments

- **Solace**: Using Solace PubSub+ Go client
  - Direct messaging (fire-and-forget)
  - 5-second publish timeout
  - Requires message VPN configuration
  - Suitable for enterprise deployments

Both publishers implement the `MessagePublisher` interface and follow the same topic pattern: `{topicPrefix}/{topicSuffix}`. The publisher is selected at startup via configuration, and only one can be enabled at a time (enforced by configuration validation).

### Communication Protocols

- **Epever**: Modbus RTU over serial (via `lumberbarons/modbus`)
  - Per-read timeout: 3 seconds
  - Retry attempts: 2 with 1-second delay between retries
  - Collection overlap prevention via mutex guard
  - 50ms delays between metric reads to prevent device lockups

### Web Server

- **Framework**: Gin (Go web framework)
- **Endpoints**:
  - `/metrics` - Prometheus metrics
  - `/api/{controller}/metrics` - JSON metrics for each controller
  - `/api/{controller}/config` - Configuration endpoints (GET/PATCH)
  - `/*` - Embedded React SPA (via `//go:embed site/build`)
- **SPA Support**: NoRoute handler serves index.html for client-side routing (React Router)

The React frontend is embedded into the binary at build time and served statically by Gin. The frontend build artifacts are copied from `site/build` to `internal/static/build` during the build process, where they're embedded using `//go:embed`.

### Configuration

YAML-based configuration with the following structure:

```yaml
solarController:
  httpPort: 8080
  debug: false  # Enable debug logging (can also use -debug flag)
  mqtt:
    enabled: true  # Only one of mqtt or solace can be enabled
    host: mqtt://broker:1883
    username: user
    password: pass
    topicPrefix: solar/metrics
  solace:
    enabled: false  # Mutually exclusive with mqtt
    host: tcp://solace-broker:55555
    username: user
    password: pass
    vpnName: default
    topicPrefix: solar/metrics
  epever:
    enabled: true
    serialPort: /dev/ttyXRUSB0
    publishPeriod: 60
```

The controller can be explicitly enabled or disabled via the `enabled` boolean field. If `enabled: false`, the controller will not start regardless of other configuration. If `enabled: true` but required fields are missing (serialPort for epever), a warning will be logged and the controller will not start.

**Message Publisher Configuration:**
- Only one of `mqtt` or `solace` can be enabled at a time
- Configuration validation enforces this mutual exclusion
- If neither is enabled, metrics are still collected and exposed via Prometheus/HTTP but not published to a message broker
- Required fields for MQTT: `host`, `topicPrefix`
- Required fields for Solace: `host`, `vpnName`, `topicPrefix`

Debug mode can be enabled via the `debug` configuration field or the `-debug` command-line flag. The command-line flag takes precedence over the config file setting.

## Project Structure

- `cmd/controller/` - Main application entry point
- `internal/controllers/` - Hardware controller implementations (epever)
- `internal/mqtt/` - MQTT publishing functionality
- `internal/solace/` - Solace publishing functionality
- `internal/publishers/` - Publisher factory and abstraction layer
- `internal/static/` - Static file embedding (React frontend)
- `site/` - React frontend source code
- `package/` - Packaging files for system packages (deb, rpm, etc.)

## Important Notes

- The React app must be built before building the Go binary (since it's embedded via `//go:embed`)
- The build process copies `site/build` to `internal/static/build` where it's embedded into the binary
- Main package is in `cmd/controller/`, not at the project root
- Controllers implement graceful shutdown via `defer controller.Close()` in `main.go`
- Message publishing is optional - if neither MQTT nor Solace is enabled, a no-op publisher is used
- Publishers implement the `MessagePublisher` interface for easy testing and swapping
- Only one message publisher (MQTT or Solace) can be enabled at a time
- The application uses structured logging via `logrus`
- All controllers register their own HTTP endpoints via `RegisterEndpoints()`
- Always add unit tests for new code
- Alaways run linters after code changes