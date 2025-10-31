# solar-controller

A Go-based service that collects metrics from solar power equipment (Epever, Victron) and publishes them via MQTT and Prometheus. It includes a React-based web UI for monitoring.

## Features

- Modular controller architecture supporting multiple hardware types
- Prometheus metrics export
- MQTT publishing for integration with home automation systems
- Web-based monitoring UI
- RESTful API for metrics and configuration
- Multi-platform builds via goreleaser

## Development

### Prerequisites

- Go 1.x or later
- Node.js and npm (for frontend development)
- Make (recommended)

### Quick Start

```bash
# Clone the repository
git clone <repository-url>
cd solar-controller

# Build everything (frontend + backend)
make build

# Run with configuration
./bin/solar-controller -config config.yaml
```

### Building

#### Using Make (Recommended)

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

#### Backend (Go)

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

**Important:** The React app must be built before building the Go binary since the frontend is embedded via `//go:embed`.

#### Frontend (React)

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

## Configuration

Create a YAML configuration file with the following structure:

```yaml
solarController:
  httpPort: 8080
  mqtt:
    host: mqtt://broker:1883
    username: user
    password: pass
    topicPrefix: solar/metrics
  epever:
    enabled: true
    serialPort: /dev/ttyXRUSB0
    publishPeriod: 60
  victron:
    enabled: true
    macAddress: AA:BB:CC:DD:EE:FF
    publishPeriod: 30
```

### Enabling/Disabling Controllers

Each controller (epever, victron) has an `enabled` boolean field that controls whether it runs:

- Set `enabled: true` to activate the controller
- Set `enabled: false` to disable the controller
- If `enabled: true` but required fields are missing (serialPort for epever, macAddress for victron), a warning will be logged and the controller will not start

MQTT publishing is optional - if no host is configured, the application will run without MQTT support.

## Architecture

### Controller Pattern

The application uses a plugin-style controller architecture where each hardware type implements the `SolarController` interface:

```go
type SolarController interface {
    RegisterEndpoints(r *gin.Engine)
    Enabled() bool
}
```

Each controller (epever, victron) follows the same structure:
- **Controller**: Main orchestrator that manages scheduled collection/publishing
- **Collector**: Handles device communication and metric collection
- **Configurer**: Manages device configuration (epever only)
- **PrometheusCollector**: Exposes metrics to Prometheus

### Communication Protocols

- **Epever**: Modbus RTU over serial (via `goburrow/modbus`)
- **Victron**: Bluetooth LE (via `rigado/ble`)

### API Endpoints

- `/metrics` - Prometheus metrics
- `/api/{controller}/metrics` - JSON metrics for each controller
- `/api/{controller}/config` - Configuration endpoints (GET/PATCH)
- `/*` - Embedded React SPA

## Project Structure

- `cmd/controller/` - Main application entry point
- `internal/controllers/` - Hardware controller implementations (epever, victron)
- `internal/publisher/` - MQTT publishing functionality
- `internal/static/` - Static file embedding (React frontend)
- `site/` - React frontend source code
- `package/` - Packaging files for system packages (deb, rpm, etc.)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
