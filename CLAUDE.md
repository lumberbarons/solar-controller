# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Solar-controller is a Go-based service that collects metrics from solar power equipment (Epever, Victron) and battery management systems (PiJuice HAT) and publishes them via MQTT and Prometheus. It includes a React-based web UI for monitoring.

## Development Commands

### Using Make (Recommended)

```bash
# Build everything (frontend + backend)
make build

# Build only frontend (React app)
make build-frontend

# Build only backend (Go binary)
make build-backend

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

Each controller (epever, victron, pijuice) follows the same structure:
- **Controller**: Main orchestrator that manages scheduled collection/publishing
- **Collector**: Handles device communication and metric collection
- **Configurer**: Manages device configuration (epever, pijuice only)
- **PrometheusCollector**: Exposes metrics to Prometheus

Controllers are instantiated in `main.go:buildControllers()` and conditionally enabled based on configuration. If required config fields are missing, the controller returns an empty/disabled instance.

### Data Flow

1. Controllers are initialized with YAML configuration at startup
2. Each enabled controller schedules periodic collection via `gocron`
3. On each tick, the controller's `collectAndPublish()` method:
   - Calls the Collector to fetch device metrics
   - Updates Prometheus metrics via PrometheusCollector
   - Publishes JSON payload to MQTT via MqttPublisher
   - Caches last status for HTTP API endpoints

### Communication Protocols

- **Epever**: Modbus RTU over serial (via `goburrow/modbus`)
- **Victron**: Bluetooth LE (via `rigado/ble`)
- **PiJuice**: I2C bus communication

### Web Server

- **Framework**: Gin (Go web framework)
- **Endpoints**:
  - `/metrics` - Prometheus metrics
  - `/api/{controller}/metrics` - JSON metrics for each controller
  - `/api/{controller}/config` - Configuration endpoints (GET/PATCH)
  - `/*` - Embedded React SPA (via `//go:embed site/build`)

The React frontend is embedded into the binary at build time and served statically by Gin. The frontend build artifacts are copied from `site/build` to `internal/static/build` during the build process, where they're embedded using `//go:embed`.

### Configuration

YAML-based configuration with the following structure:

```yaml
solarController:
  httpPort: 8080
  mqtt:
    host: mqtt://broker:1883
    username: user
    password: pass
    topicPrefix: solar/metrics
  epever:
    serialPort: /dev/ttyXRUSB0
    publishPeriod: 60
  victron:
    macAddress: AA:BB:CC:DD:EE:FF
    publishPeriod: 30
  pijuice:
    i2cBus: 1
    i2cAddress: 0x14
    publishPeriod: 30
```

Controllers are disabled if their required config fields are empty (serialPort, macAddress, i2cAddress).

## Project Structure

- `cmd/controller/` - Main application entry point
- `internal/controllers/` - Hardware controller implementations (epever, victron, pijuice)
- `internal/publisher/` - MQTT publishing functionality
- `internal/static/` - Static file embedding (React frontend)
- `site/` - React frontend source code
- `package/` - Packaging files for system packages (deb, rpm, etc.)

## Important Notes

- The React app must be built before building the Go binary (since it's embedded via `//go:embed`)
- The build process copies `site/build` to `internal/static/build` where it's embedded into the binary
- Main package is in `cmd/controller/`, not at the project root
- Controllers implement graceful shutdown via `defer controller.Close()` in `main.go`
- MQTT publishing is optional - if no host is configured, MqttPublisher returns an empty/no-op instance
- The application uses structured logging via `logrus`
- All controllers register their own HTTP endpoints via `RegisterEndpoints()`
