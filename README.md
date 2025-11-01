# solar-controller

A Go-based service that collects metrics from solar power equipment (Epever) and publishes them via MQTT and Prometheus. It includes a React-based web UI for monitoring.

## Features

- Modular controller architecture supporting multiple hardware types
- Prometheus metrics export
- MQTT publishing for integration with home automation systems
- Web-based monitoring UI
- RESTful API for metrics and configuration
- Multi-platform builds via goreleaser

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

# Build backend for Linux ARM64
make build-linux-arm64

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
  debug: false  # Enable debug logging (can also use -debug flag)
  mqtt:
    enabled: true
    host: mqtt://broker:1883
    username: user
    password: pass
    topicPrefix: solar/metrics
  epever:
    enabled: true
    serialPort: /dev/ttyXRUSB0
    publishPeriod: 60
```

### Enabling/Disabling Controllers

Each controller (epever) has an `enabled` boolean field that controls whether it runs:

- Set `enabled: true` to activate the controller
- Set `enabled: false` to disable the controller
- If `enabled: true` but required fields are missing (serialPort for epever), a warning will be logged and the controller will not start

### Enabling/Disabling MQTT

MQTT publishing has an `enabled` boolean field that controls whether it runs:

- Set `enabled: true` to activate MQTT publishing
- Set `enabled: false` to disable MQTT publishing
- If `enabled: true` but required fields are missing (host or topicPrefix), configuration validation will fail

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
- `GET /api/solar/metrics` - JSON metrics for Epever controller (current status)

#### Configuration Endpoints
- `GET /api/solar/battery-profile` - Get battery type and capacity
- `PATCH /api/solar/battery-profile` - Update battery type and/or capacity
- `GET /api/solar/charging-parameters` - Get all charging parameters (voltages, durations, temperature limits)
- `PATCH /api/solar/charging-parameters` - Update charging parameters (only when battery type is 'userDefined')
- `GET /api/solar/time` - Get controller's current time
- `PATCH /api/solar/time` - Set controller's time

#### Legacy Endpoints (for backwards compatibility)
- `GET /api/solar/config` - Get all configuration settings
- `PATCH /api/solar/config` - Update configuration settings

#### Frontend
- `/*` - Embedded React SPA for web-based monitoring

### API Usage Examples

#### Get current metrics
```bash
curl http://localhost:8080/api/solar/metrics
```

#### Get battery profile
```bash
curl http://localhost:8080/api/solar/battery-profile
```

#### Update battery type
```bash
curl -X PATCH http://localhost:8080/api/solar/battery-profile \
  -H "Content-Type: application/json" \
  -d '{"batteryType": "userDefined"}'
```

**Valid battery types**: `userDefined`, `sealed`, `gel`, `flooded`

#### Update charging parameters
```bash
curl -X PATCH http://localhost:8080/api/solar/charging-parameters \
  -H "Content-Type: application/json" \
  -d '{"boostVoltage": 14.4, "floatVoltage": 13.8}'
```

**Note**: Charging parameters can only be modified when battery type is set to 'userDefined'. The API validates voltage relationships to ensure safe charging parameters.

## Project Structure

- `cmd/controller/` - Main application entry point
- `internal/controllers/` - Hardware controller implementations (epever)
- `internal/mqtt/` - MQTT publishing functionality
- `internal/static/` - Static file embedding (React frontend)
- `site/` - React frontend source code
- `package/` - Packaging files for system packages (deb, rpm, etc.)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
