# solar-controller

A Go-based service that collects metrics from solar power equipment (Epever) and publishes them via multiple backends (MQTT, Solace, File, or Prometheus Remote Write). Metrics are also exposed via Prometheus scraping endpoint. It includes a React-based web UI for monitoring.

## Features

- Modular controller architecture supporting multiple hardware types
- Multiple publishing options:
  - **MQTT** - Lightweight message broker for home automation systems
  - **Solace** - Enterprise-grade messaging with Solace PubSub+
  - **File** - JSON log files with automatic rotation and optional compression
  - **Prometheus Remote Write** - Push metrics to Prometheus, Grafana Cloud, VictoriaMetrics, etc.
- Prometheus metrics export via scraping endpoint
- Web-based monitoring UI (React SPA)
- RESTful API for metrics and configuration
- Multi-platform builds (amd64/arm64) with native compilation
- Docker images for both amd64 and arm64 architectures
- Debian and RPM packages for easy deployment

## Development

### Prerequisites

- Go 1.24.0 or later
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
# Show available make targets
make help

# Build everything (frontend + backend)
make build

# Build only frontend (React app)
make build-frontend

# Build only backend (Go binary)
make build-backend

# Build backend for Linux ARM64 using Docker
make build-linux-arm64-docker

# Build Docker image
make docker

# Deploy to remote server (requires REMOTE_HOST)
make deploy REMOTE_HOST=user@host

# Run tests
make test

# Clean build artifacts
make clean
```

#### Backend (Go)

```bash
# Build the application with CGO enabled (requires frontend to be built first)
CGO_ENABLED=1 go build -o bin/solar-controller ./cmd/controller

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

**Important Notes:**
- The React app must be built before building the Go binary since the frontend is embedded via `//go:embed`
- CGO is required for Solace messaging support. Use `CGO_ENABLED=1` when building with Solace enabled
- The Makefile automatically enables CGO when using `make build-backend`

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

The project uses GitHub Actions for multi-platform releases with native builds:

```bash
# Create a release by pushing a tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

The release workflow automatically:
- Builds binaries natively on architecture-specific runners (amd64 and arm64) with CGO enabled
- Creates .deb and .rpm packages using nfpm
- Builds Docker images for both amd64 and arm64 architectures
- Creates a GitHub release with all artifacts

Releases include:
- Standalone binaries for Linux (amd64/arm64) and macOS (amd64/arm64)
- Debian packages (.deb)
- RPM packages (.rpm)
- Multi-architecture Docker images

## Configuration

Create a YAML configuration file with the following structure:

```yaml
solarController:
  httpPort: 8080
  debug: false          # Enable debug logging (can also use -debug flag)
  deviceId: controller-123      # Unique identifier for this device (default: "controller-1")
  topicPrefix: solar            # Topic prefix for all publishers (default: "solar")

  # Message Publishers (choose one - mutually exclusive)
  mqtt:
    enabled: true  # Only one of mqtt, solace, file, or remoteWrite can be enabled
    host: mqtt://broker:1883
    username: user
    password: pass

  solace:
    enabled: false  # Enterprise messaging with Solace PubSub+
    host: tcp://solace-broker:55555
    username: user
    password: pass
    vpnName: default

  file:
    enabled: false  # Write metrics to rotating log files
    filename: /var/log/solar-controller/metrics.log
    maxSizeMB: 10      # Max size per file before rotation (default: 10)
    maxBackups: 10     # Number of old files to keep (default: 10)
    compress: false    # Compress rotated files with gzip (default: false)

  remoteWrite:
    enabled: false  # Push to Prometheus, Grafana Cloud, VictoriaMetrics, etc.
    url: http://prometheus:9090/api/v1/write
    timeout: 30s    # Optional (default: 30s)
    basicAuth:      # Optional (mutually exclusive with bearerToken)
      username: user
      password: pass
    bearerToken: token123  # Optional (mutually exclusive with basicAuth)
    headers:        # Optional custom headers
      X-Scope-OrgID: tenant1

  # Hardware Controllers
  epever:
    enabled: true
    serialPort: /dev/ttyXRUSB0
    publishPeriod: 60
```

### Configuration Details

**Hardware Controllers:**
- Each controller (e.g., epever) has an `enabled` boolean field
- Set `enabled: true` to activate the controller
- If required fields are missing (serialPort for epever), a warning will be logged and the controller won't start

**Message Publishers:**
- Only one publisher (mqtt, solace, file, or remoteWrite) can be enabled at a time
- If multiple publishers are enabled, configuration validation will fail
- If no publisher is enabled, metrics are still collected and exposed via Prometheus/HTTP but not published
- The `deviceId` and `topicPrefix` are global settings used by all publishers

**Publisher Options:**
- **MQTT**: Lightweight message broker integration (QoS 0, 5s timeout)
- **Solace**: Enterprise messaging with VPN support (direct messaging, 5s timeout)
- **File**: JSON log files with rotation, compression, and configurable retention
- **Remote Write**: Push to Prometheus-compatible endpoints with authentication and custom headers

### Debug Mode

Debug mode can be enabled in two ways:

- **Config file**: Set `debug: true` in the configuration file
- **Command-line flag**: Use the `-debug` flag when running the application

The command-line flag takes precedence over the config file setting. When debug mode is enabled, the application will output detailed logging information including modbus register reads, collection timing, and operational details.

## Testing

### Testing with Modbus Simulator

You can test the solar controller without physical hardware using the Modbus simulator from the [lumberbarons/modbus](https://github.com/lumberbarons/modbus) project.

#### Prerequisites

1. Clone and build the modbus simulator:
   ```bash
   git clone https://github.com/lumberbarons/modbus.git
   cd modbus
   go build -o bin/modbus-simulator ./cmd/simulator
   ```

#### Running the Simulator

1. Start the Modbus simulator with the solar charger configuration:
   ```bash
   ./bin/modbus-simulator --config /path/to/solar-controller/testdata/simulator/epever.json --mode rtu --baud 115200
   ```

2. Configure solar-controller to use the virtual port created by the simulator:
   ```yaml
   solarController:
     httpPort: 8080
     mqtt:
       enabled: false
     epever:
       enabled: true
       serialPort: /dev/ttys003  # the virtual port in the simulator's logs
       publishPeriod: 60
   ```

3. Run solar-controller:
   ```bash
   ./bin/solar-controller -config config.yaml
   ```

The simulator provides realistic Epever solar charge controller data including:

- PV voltage, current, and power readings
- Battery voltage, power, temperature, and state of charge
- Load voltage, current, and power
- Equipment temperature
- Battery configuration parameters (type, capacity, voltage thresholds)

You can modify `testdata/simulator/solar-charger.json` to simulate different device states and values.

## Architecture

### Controller Pattern

The application uses a plugin-style controller architecture where each hardware type implements the `SolarController` interface:

```go
type SolarController interface {
    RegisterEndpoints(r *gin.Engine)
    Enabled() bool
}
```

Each controller follows the same structure:
- **Controller**: Main orchestrator that manages scheduled collection/publishing
- **Collector**: Handles device communication and metric collection
- **Configurer**: Manages device configuration
- **PrometheusCollector**: Exposes metrics to Prometheus

### Communication Protocols

- **Epever**: Modbus RTU over serial (via `lumberbarons/modbus`)

#### Modbus Communication Reliability

The controller implements several strategies to ensure reliable communication with solar charge controllers:

**Timeouts and Retries:**
- **Per-read timeout**: 3 seconds for each individual register read operation
- **Retry attempts**: Up to 2 attempts with 1-second delay between retries
- **Collection timeout**: 30-second overall timeout for the entire collection cycle
- **Collection overlap prevention**: A mutex guard prevents overlapping collection cycles if a previous collection is still running

**Inter-request Delays:**
- **Configuration reads** (holding registers): 75-100ms delays between operations
- **Metrics collection** (input registers): 50ms delays between operations
- **Write operations**: 150ms delay after each write, plus 500ms settling time before read-back
- All modbus operations are serialized through a mutex to prevent concurrent access

These delays and timeouts ensure the Epever device has adequate time to process EEPROM operations and prevent communication lockups, especially when reading configuration data or writing multiple parameters.

### API Endpoints

#### Monitoring Endpoints
- `GET /metrics` - Prometheus metrics export
- `GET /api/epever/metrics` - JSON metrics for Epever controller (current status)

#### Configuration Endpoints
- `GET /api/epever/battery-profile` - Get battery type and capacity
- `PATCH /api/epever/battery-profile` - Update battery type and/or capacity
- `GET /api/epever/charging-parameters` - Get all charging parameters (voltages, durations, temperature limits)
- `PATCH /api/epever/charging-parameters` - Update charging parameters (only when battery type is 'userDefined')
- `GET /api/epever/time` - Get controller's current time
- `PATCH /api/epever/time` - Set controller's time

#### Legacy Endpoints (for backwards compatibility)
- `GET /api/epever/config` - Get all configuration settings
- `PATCH /api/epever/config` - Update configuration settings

#### Frontend
- `/*` - Embedded React SPA for web-based monitoring

### API Usage Examples

#### Get current metrics
```bash
curl http://localhost:8080/api/epever/metrics
```

#### Get battery profile
```bash
curl http://localhost:8080/api/epever/battery-profile
```

#### Update battery type
```bash
curl -X PATCH http://localhost:8080/api/epever/battery-profile \
  -H "Content-Type: application/json" \
  -d '{"batteryType": "userDefined"}'
```

**Valid battery types**: `userDefined`, `sealed`, `gel`, `flooded`

#### Update charging parameters
```bash
curl -X PATCH http://localhost:8080/api/epever/charging-parameters \
  -H "Content-Type: application/json" \
  -d '{"boostVoltage": 14.4, "floatVoltage": 13.8}'
```

**Note**: Charging parameters can only be modified when battery type is set to 'userDefined'. The API validates voltage relationships to ensure safe charging parameters.

## Message Publishing

The application supports flexible message publishing with four backend options:

### Topic Structure

Messages are published with one message per metric using this pattern:
```
{topicPrefix}/{deviceId}/{controller}/{metric-name}
```

Example with `topicPrefix: "solar"` and `deviceId: "controller-123"`:
```
solar/controller-123/epever/array-voltage
solar/controller-123/epever/battery-soc
solar/controller-123/epever/charging-power
```

### Message Payload

Each metric message contains JSON with value, unit, and timestamp:
```json
{
  "value": 18.5,
  "unit": "volts",
  "timestamp": 1699000000
}
```

### Prometheus Remote Write

When using the RemoteWrite publisher, metrics are converted to Prometheus format:
- Metric names: `{controller}_{metric_name}` (snake_case)
- Labels: `device_id`, `controller`, `unit`
- Example: `epever_battery_voltage{device_id="controller-123",unit="volts"}`

All metrics from each collection cycle are batched into a single WriteRequest for efficiency.

### Testing Remote Write

To test Prometheus remote_write locally:
```bash
# Start VictoriaMetrics test server
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
- `internal/file/` - File publishing with log rotation
- `internal/remotewrite/` - Prometheus remote_write publishing
- `internal/publishers/` - Publisher factory and abstraction
- `internal/static/` - Static file embedding (React frontend)
- `site/` - React frontend source code
- `testing/` - Remote write testing setup and utilities
- `package/` - Packaging files for system packages (deb, rpm, etc.)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
