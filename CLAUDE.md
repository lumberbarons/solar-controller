# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Solar-controller is a Go-based service that collects metrics from solar power equipment (Epever) and publishes them via MQTT, Solace, file logging, or Prometheus remote_write, with metrics also exposed via Prometheus scraping. It includes a React-based web UI for monitoring.

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

# Build with CGO enabled (required for Solace support)
make build-with-cgo

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

**Note:** The Solace messaging library requires CGO to be enabled. When building locally with Solace support, use `make build-with-cgo` or set `CGO_ENABLED=1` when running `go build`.

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

The project uses GitHub Actions for multi-platform releases with native builds on split runners:

```bash
# Create a release by pushing a tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

The release workflow:
- Builds binaries natively on architecture-specific runners (CGO enabled)
- Creates .deb and .rpm packages using nfpm
- Builds Docker images for both amd64 and arm64
- Creates a GitHub release with all artifacts

**Note:** CGO is required for the Solace messaging library, so all builds must be done on native architecture runners or with appropriate cross-compilation toolchains.

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
   - Publishes individual metric messages to message broker (MQTT or Solace) via MessagePublisher interface
   - Caches last status for HTTP API endpoints

### Message Publishing

The application supports four message publisher options (mutually exclusive):

- **MQTT**: Using Eclipse Paho MQTT client
  - QoS 0 (fire-and-forget)
  - 5-second publish timeout
  - Suitable for lightweight deployments

- **Solace**: Using Solace PubSub+ Go client
  - Direct messaging (fire-and-forget)
  - 5-second publish timeout
  - Requires message VPN configuration
  - Suitable for enterprise deployments

- **File**: Using lumberjack for log rotation
  - Writes JSON-formatted metrics to rotating log files
  - Configurable max file size (default: 10MB)
  - Configurable max backup files (default: 10)
  - Optional compression of rotated files
  - Suitable for offline logging, archival, or edge deployments with intermittent connectivity

- **Prometheus Remote Write**: Using official Prometheus remote_write protocol (v1.0)
  - Batches all metrics from each collection cycle into a single WriteRequest
  - Snappy-compressed protobuf over HTTP/HTTPS
  - Supports HTTP Basic Auth and Bearer Token authentication
  - Custom headers support (e.g., X-Scope-OrgID for multi-tenancy)
  - Configurable timeout (default: 30s)
  - Suitable for pushing metrics to Prometheus, Cortex, VictoriaMetrics, Grafana Cloud, etc.

All publishers implement the `MessagePublisher` interface. The publisher is selected at startup via configuration, and only one can be enabled at a time (enforced by configuration validation).

#### Topic Structure

Messages are published with one message per metric using the following topic pattern:

```
{topicPrefix}/{deviceId}/{controller}/{metric-name}
```

For example, with configuration `topicPrefix: "solar"` and `deviceId: "controller-123"`:
```
solar/controller-123/epever/array-voltage
solar/controller-123/epever/battery-soc
solar/controller-123/epever/charging-power
```

#### Message Payload

Each metric message contains a JSON payload with the metric value, unit, and timestamp:

```json
{
  "value": 18.5,
  "unit": "volts",
  "timestamp": 1699000000
}
```

#### Metric Names and Units

Epever controller publishes the following metrics (kebab-case naming):

- `array-voltage` (volts) - Solar panel voltage
- `array-current` (amperes) - Solar panel current
- `array-power` (watts) - Solar panel power
- `charging-current` (amperes) - Battery charging current
- `charging-power` (watts) - Battery charging power
- `battery-voltage` (volts) - Battery voltage
- `battery-soc` (percent) - Battery state of charge
- `battery-temp` (celsius) - Battery temperature
- `device-temp` (celsius) - Controller device temperature
- `energy-generated-daily` (kilowatt-hours) - Daily energy generation
- `charging-status` (code) - Charging status code
- `collection-time` (seconds) - Time taken to collect metrics

#### Wildcard Subscriptions

MQTT/Solace subscribers can use wildcard patterns:
- `solar/+/epever/#` - All epever metrics from all devices
- `solar/controller-123/epever/#` - All metrics from specific device
- `solar/controller-123/epever/battery-+` - All battery-related metrics

#### Prometheus Remote Write Metric Naming

When using the RemoteWrite publisher, metrics are converted from the topic-based format to Prometheus metric naming conventions:

**Naming Convention:**
- Topic format: `{deviceId}/{controller}/{metric-name}` (kebab-case)
- Prometheus metric name: `{controller}_{metric_name}` (snake_case)
- Example: `controller-123/epever/battery-voltage` → `epever_battery_voltage`

**Labels:**
- `__name__`: The metric name (e.g., `epever_battery_voltage`)
- `device_id`: Device identifier from configuration (e.g., `controller-123`)
- `controller`: Controller type (e.g., `epever`)
- `unit`: Unit of measurement from metric payload (e.g., `volts`, `amperes`, `percent`)

**Example Prometheus Query:**
```promql
# Get battery voltage for a specific device
epever_battery_voltage{device_id="controller-123"}

# Get all metrics from epever controllers
{controller="epever"}

# Get all voltage metrics across all devices
{unit="volts"}
```

**Batching:**
All 12 metrics from each collection cycle are batched into a single WriteRequest, reducing HTTP overhead and improving efficiency.

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
  - `/api/epever/metrics` - JSON metrics for Epever controller
  - `/api/epever/battery-profile` - Battery profile configuration (GET/PATCH)
  - `/api/epever/charging-parameters` - Charging parameters configuration (GET/PATCH)
  - `/api/epever/time` - Controller time (GET/PATCH)
  - `/api/epever/config` - Legacy configuration endpoint (GET/PATCH)
  - `/*` - Embedded React SPA (via `//go:embed site/build`)
- **SPA Support**: NoRoute handler serves index.html for client-side routing (React Router)
- **Namespace**: Each controller registers endpoints under `/api/{controllerName}` where the controller name matches the hardware type (e.g., "epever")

The React frontend is embedded into the binary at build time and served statically by Gin. The frontend build artifacts are copied from `site/build` to `internal/static/build` during the build process, where they're embedded using `//go:embed`.

### Configuration

YAML-based configuration with the following structure:

```yaml
solarController:
  httpPort: 8080
  debug: false          # Enable debug logging (can also use -debug flag)
  deviceId: controller-123      # Unique identifier for this device (default: "controller-1")
  topicPrefix: solar            # Topic prefix for all publishers (default: "solar")
  mqtt:
    enabled: true  # Only one of mqtt, solace, file, or remoteWrite can be enabled
    host: mqtt://broker:1883
    username: user
    password: pass
  solace:
    enabled: false  # Mutually exclusive with mqtt, file, and remoteWrite
    host: tcp://solace-broker:55555
    username: user
    password: pass
    vpnName: default
  file:
    enabled: false  # Mutually exclusive with mqtt, solace, and remoteWrite
    filename: /var/log/solar-controller/metrics.log
    maxSizeMB: 10      # Max size per file before rotation (default: 10)
    maxBackups: 10     # Number of old files to keep (default: 10)
    compress: false    # Compress rotated files with gzip (default: false)
  remoteWrite:
    enabled: false  # Mutually exclusive with mqtt, solace, and file
    url: http://prometheus:9090/api/v1/write  # Required when enabled
    timeout: 30s    # Optional (default: 30s)
    basicAuth:      # Optional (mutually exclusive with bearerToken)
      username: user
      password: pass
    bearerToken: token123  # Optional (mutually exclusive with basicAuth)
    headers:        # Optional custom headers
      X-Scope-OrgID: tenant1
  epever:
    enabled: true
    serialPort: /dev/ttyXRUSB0
    publishPeriod: 60
```

The controller can be explicitly enabled or disabled via the `enabled` boolean field. If `enabled: false`, the controller will not start regardless of other configuration. If `enabled: true` but required fields are missing (serialPort for epever), a warning will be logged and the controller will not start.

**Global Configuration:**
- `deviceId` (optional): Unique identifier for this device instance, used in publisher topics across all controllers. Defaults to `"controller-1"` if not specified.
- `topicPrefix` (optional): Topic prefix prepended to all published messages. Used by all publisher types (MQTT, Solace, File). Defaults to `"solar"` if not specified.
- `httpPort` (required): HTTP server port (1-65535)
- `debug` (optional): Enable debug logging, can also be set via `-debug` command-line flag

**Epever Controller Configuration:**
- `serialPort` (required): Serial port path for Modbus RTU communication
- `publishPeriod` (required): Collection interval in seconds

**Message Publisher Configuration:**
- Only one of `mqtt`, `solace`, `file`, or `remoteWrite` can be enabled at a time
- Configuration validation enforces this mutual exclusion
- If none is enabled, metrics are still collected and exposed via Prometheus/HTTP but not published
- Global `topicPrefix` is used by MQTT, Solace, and File publishers (defaults to `"solar"`)
- Required fields for MQTT: `host`
- Required fields for Solace: `host`, `vpnName`
- Required fields for File: `filename`
- Optional fields for File: `maxSizeMB` (default: 10), `maxBackups` (default: 10), `compress` (default: false)
- Required fields for RemoteWrite: `url`
- Optional fields for RemoteWrite: `timeout` (default: 30s), `basicAuth`, `bearerToken`, `headers`
- RemoteWrite authentication: `basicAuth` and `bearerToken` are mutually exclusive

Debug mode can be enabled via the `debug` configuration field or the `-debug` command-line flag. The command-line flag takes precedence over the config file setting.

### Testing Remote Write

To test Prometheus remote_write locally without Grafana Cloud:

```bash
# Start VictoriaMetrics
cd testing && ./test-remotewrite.sh

# In another terminal, run solar-controller
make build-backend
./bin/solar-controller -config testing/config-remotewrite-test.yaml

# View metrics at http://localhost:8428/vmui
```

See `testing/README.md` for details.

## Project Structure

- `cmd/controller/` - Main application entry point
- `internal/controllers/` - Hardware controller implementations (epever)
- `internal/mqtt/` - MQTT publishing functionality
- `internal/solace/` - Solace publishing functionality
- `internal/file/` - File publishing functionality with log rotation
- `internal/remotewrite/` - Prometheus remote_write publishing functionality
- `internal/publishers/` - Publisher factory and abstraction layer
- `internal/static/` - Static file embedding (React frontend)
- `site/` - React frontend source code
- `testing/` - Remote write testing setup and utilities
- `package/` - Packaging files for system packages (deb, rpm, etc.)

## Important Notes

- The React app must be built before building the Go binary (since it's embedded via `//go:embed`)
- The build process copies `site/build` to `internal/static/build` where it's embedded into the binary
- Main package is in `cmd/controller/`, not at the project root
- Controllers implement graceful shutdown via `defer controller.Close()` in `main.go`
- Message publishing is optional - if no publisher (MQTT, Solace, File, or RemoteWrite) is enabled, a no-op publisher is used
- Publishers implement the `MessagePublisher` interface for easy testing and swapping
- Only one message publisher (MQTT, Solace, File, or RemoteWrite) can be enabled at a time
- The application uses structured logging via `logrus`
- All controllers register their own HTTP endpoints via `RegisterEndpoints()`
- Always add unit tests for new code
- Alaways run linters after code changes