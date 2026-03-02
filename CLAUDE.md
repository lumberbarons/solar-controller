# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Solar-controller is a Go-based service that collects metrics from solar power equipment (Epever) and publishes them via MQTT, Solace, AWS SNS, file logging, or Prometheus remote_write, with metrics also exposed via Prometheus scraping. It includes a React-based web UI for monitoring.

## Project Navigation

| Directory | What | When to read |
|-----------|------|--------------|
| `cmd/controller/` | Application entry point | Changing startup, flags, or controller wiring |
| `internal/controllers/` | Hardware controller implementations (epever) | Adding controllers, modifying collection or publishing logic |
| `internal/publishers/` | Publisher factory, MultiPublisher, MessagePublisher interface | Adding a new publisher or changing fan-out behavior |
| `internal/mqtt/` | MQTT publisher | Modifying MQTT publishing |
| `internal/solace/` | Solace publisher | Modifying Solace publishing |
| `internal/sns/` | AWS SNS publisher | Modifying SNS publishing |
| `internal/file/` | File publisher with log rotation | Modifying file publishing |
| `internal/remotewrite/` | Prometheus remote_write publisher | Modifying remote write publishing |
| `internal/config/` | YAML configuration structs and validation | Changing configuration options |
| `internal/app/` | Application bootstrap and HTTP server | Changing server setup or middleware |
| `internal/static/` | Embedded React frontend (`//go:embed`) | Changing how the frontend is served |
| `internal/testing/` | Integration test container helpers | Adding integration tests |
| `site/` | React frontend source | Modifying the web UI |
| `testing/` | Remote write test setup (VictoriaMetrics) | Testing remote write locally |
| `testdata/` | Modbus simulator configuration | Testing with simulated hardware |
| `docs/` | Modbus register documentation | Understanding Epever register mappings |
| `package/` | System packaging (deb, rpm) via nfpm | Changing release packaging |
| `Makefile` | Build, test, deploy orchestration | Modifying build targets or CI commands |
| `Dockerfile` | Production container image | Changing container build or runtime |
| `Dockerfile.build` | Cross-compilation build container | Changing ARM64 cross-build process |
| `nfpm.yaml` | Package metadata (deb, rpm) | Changing package version, dependencies, or scripts |
| `go.mod` | Go module and dependency versions | Adding or updating dependencies |

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

### Backend (Go)

```bash
# Build the application with CGO enabled (requires frontend to be built first)
CGO_ENABLED=1 go build -o bin/solar-controller ./cmd/controller

# Run with configuration
./bin/solar-controller -config path/to/config.yaml

# Run in debug mode
./bin/solar-controller -config path/to/config.yaml -debug

# Run unit tests
go test ./...

# Run unit tests (via Makefile)
make test-unit

# Run integration tests (requires Docker)
make test-int

# Run all tests (unit + integration)
make test-all

# Install dependencies
go get -d -v ./...

# Tidy dependencies
go mod tidy
```

**Note:** The Solace messaging library requires CGO to be enabled. Use `make build-backend` (which enables CGO automatically) or set `CGO_ENABLED=1` when running `go build` directly.

**Testing:**
- **Unit tests**: Run without build tags, use mocks for external dependencies
- **Integration tests**: Require Docker, use `//go:build integration` tag, test with real services via testcontainers
- Integration tests are automatically skipped when running `make test` or `go test ./...`

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
# Build Docker image using Makefile (recommended)
make docker

# Or build manually
docker build -t solar-controller .

# Run container
docker run solar-controller -config /etc/solar-controller/config.yaml
```

**Docker Build Features:**
- Multi-stage build for minimal image size
- Supports both amd64 and arm64 architectures
- Includes all required dependencies for Solace support (CGO enabled)
- Frontend is embedded at build time

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

**Release Artifacts:**
- Standalone binaries for Linux (amd64/arm64) and macOS (amd64/arm64)
- Debian packages (.deb) for easy installation on Debian/Ubuntu systems
- RPM packages (.rpm) for RHEL/CentOS/Fedora systems
- Multi-architecture Docker images pushed to registry

**Note:** CGO is required for the Solace messaging library, so all builds must be done on native architecture runners or with appropriate cross-compilation toolchains.

### Deployment

The project includes a convenient deployment workflow:

```bash
# Build for Linux ARM64 and deploy to remote server
make deploy REMOTE_HOST=user@host
```

This command:
1. Builds the Linux ARM64 binary using Docker
2. Copies the binary to the remote server via SCP
3. Installs it to `/usr/bin` with proper permissions
4. Restarts the `solar-controller` systemd service

**Prerequisites:**
- SSH access to the remote server
- `solar-controller` systemd service configured on the remote server
- User has sudo privileges on the remote server

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

The application supports multiple message publishers that can be enabled simultaneously:

- **MQTT**: Using Eclipse Paho MQTT client
  - QoS 0 (fire-and-forget)
  - 5-second publish timeout
  - Suitable for lightweight deployments

- **Solace**: Using Solace PubSub+ Go client
  - Direct messaging (fire-and-forget)
  - 5-second publish timeout
  - Requires message VPN configuration
  - Suitable for enterprise deployments

- **AWS SNS**: Using AWS SDK for Go v2
  - Publishes messages to an SNS topic
  - 5-second publish timeout
  - Supports AWS credentials from environment, config files, or IAM roles
  - Message subject is set to the full topic path
  - Suitable for AWS-based deployments and notification workflows

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

All publishers implement the `MessagePublisher` interface. Multiple publishers can be enabled simultaneously - metrics will be published to all enabled publishers (fan-out pattern). When multiple publishers are enabled, the factory creates a `MultiPublisher` that wraps all enabled publishers and distributes messages to each one. Individual publishers continue to operate independently with best-effort delivery semantics.

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

**Normal Metrics** (published on successful collection):
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

**Failure Metrics** (published when collection fails):
- `collection-failure` (count) - Collection failure indicator (value is always 1, published when complete collection fails)

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
All 12 normal metrics from each successful collection cycle are batched into a single WriteRequest, reducing HTTP overhead and improving efficiency. When collection fails, a single `collection-failure` metric is published instead.

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
  mqtt:
    enabled: true       # Multiple publishers can be enabled simultaneously
    host: mqtt://broker:1883
    username: user
    password: pass
    topicPrefix: solar  # Topic prefix for MQTT (default: "solar")
  solace:
    enabled: false
    host: tcp://solace-broker:55555
    username: user
    password: pass
    vpnName: default
    topicPrefix: solar  # Topic prefix for Solace (default: "solar")
  sns:
    enabled: false
    region: us-east-1
    topicArn: arn:aws:sns:us-east-1:123456789012:solar-metrics
    topicPrefix: solar  # Topic prefix for SNS (default: "solar")
  file:
    enabled: false
    filename: /var/log/solar-controller/metrics.log
    maxSizeMB: 10       # Max size per file before rotation (default: 10)
    maxBackups: 10      # Number of old files to keep (default: 10)
    compress: false     # Compress rotated files with gzip (default: false)
  remoteWrite:
    enabled: false
    url: http://prometheus:9090/api/v1/write  # Required when enabled
    timeout: 30s        # Optional (default: 30s)
    basicAuth:          # Optional (mutually exclusive with bearerToken)
      username: user
      password: pass
    bearerToken: token123  # Optional (mutually exclusive with basicAuth)
    headers:            # Optional custom headers
      X-Scope-OrgID: tenant1
    topicPrefix: solar  # Topic prefix for RemoteWrite (default: "solar")
  epever:
    enabled: true
    serialPort: /dev/ttyXRUSB0
    publishPeriod: 60
```

The controller can be explicitly enabled or disabled via the `enabled` boolean field. If `enabled: false`, the controller will not start regardless of other configuration. If `enabled: true` but required fields are missing (serialPort for epever), a warning will be logged and the controller will not start.

**Global Configuration:**
- `deviceId` (optional): Unique identifier for this device instance, used in publisher topics across all controllers. Defaults to `"controller-1"` if not specified.
- `httpPort` (required): HTTP server port (1-65535)
- `debug` (optional): Enable debug logging, can also be set via `-debug` command-line flag

**Epever Controller Configuration:**
- `serialPort` (required): Serial port path for Modbus RTU communication
- `publishPeriod` (required): Collection interval in seconds

**Message Publisher Configuration:**
- Multiple publishers can be enabled simultaneously - metrics will be published to all enabled publishers
- If none is enabled, metrics are still collected and exposed via Prometheus/HTTP but not published
- MQTT, Solace, SNS, and RemoteWrite publishers have their own `topicPrefix` configuration that defaults to `"solar"` if not specified
- File publisher does not use topicPrefix - it writes the full topic path directly to the log file
- **MQTT Publisher:**
  - Required fields: `host`
  - Optional fields: `username`, `password`, `topicPrefix` (default: "solar")
- **Solace Publisher:**
  - Required fields: `host`, `vpnName`
  - Optional fields: `username`, `password`, `topicPrefix` (default: "solar")
- **SNS Publisher:**
  - Required fields: `region`, `topicArn`
  - Optional fields: `topicPrefix` (default: "solar")
  - AWS credentials are loaded from standard AWS SDK credential chain (environment variables, config files, IAM roles, etc.)
- **File Publisher:**
  - Required fields: `filename`
  - Optional fields: `maxSizeMB` (default: 10), `maxBackups` (default: 10), `compress` (default: false)
  - Note: Topics are written directly without prefix (format: `{deviceId}/{controller}/{metric-name}`)
- **RemoteWrite Publisher:**
  - Required fields: `url`
  - Optional fields: `timeout` (default: 30s), `basicAuth`, `bearerToken`, `headers`, `topicPrefix` (default: "solar")
  - RemoteWrite authentication: `basicAuth` and `bearerToken` are mutually exclusive

Debug mode can be enabled via the `debug` configuration field or the `-debug` command-line flag. The command-line flag takes precedence over the config file setting.

### Testing Remote Write

To test Prometheus remote_write locally without Grafana Cloud:

```bash
# Start VictoriaMetrics
cd testing/remotewrite && ./test-remotewrite.sh

# In another terminal, run solar-controller
make build-backend
./bin/solar-controller -config testing/remotewrite/config.yaml

# View metrics at http://localhost:8428/vmui
```

See `testing/remotewrite/README.md` for details.

### Integration Testing

The project uses [testcontainers-go](https://golang.testcontainers.org/) for integration testing with real service dependencies. Integration tests are marked with the `//go:build integration` build tag to separate them from unit tests.

#### Running Integration Tests

**Prerequisites:**
- Docker daemon running locally
- Docker socket accessible (typically `/var/run/docker.sock`)
- Sufficient Docker resources allocated

**Commands:**
```bash
# Run only integration tests
make test-int

# Run all tests (unit + integration)
make test-all

# Run integration tests directly with go test
go test -v -tags=integration ./...

# Run specific integration test
go test -v -tags=integration ./internal/sns -run TestSNSPublisherIntegration
```

#### Available Integration Tests

**SNS Publisher Integration** (`internal/sns/integration_test.go`):
- Uses LocalStack testcontainer to simulate AWS SNS
- Tests message publishing to SNS topics
- Verifies message format and delivery via SQS subscription
- Tests custom topic prefixes

**Test Coverage:**
- Single message publishing
- Multiple message batching
- Custom topic prefix configuration
- Message payload format validation
- SNS subject (topic path) verification

#### Architecture

**Helper Package** (`internal/testing/containers/`):
- `localstack.go` - LocalStack container setup and helpers
- `helpers.go` - Common test utilities

**Container Lifecycle:**
1. Test starts container using testcontainers-go
2. Container is configured with service-specific settings
3. Test runs against real service in container
4. Container is automatically cleaned up after test (via `t.Cleanup()`)

**Benefits:**
- **Production parity**: Tests run against same service images as production
- **Automated**: No manual setup required, containers start automatically
- **Isolated**: Each test gets unique port mappings for parallel execution
- **Reproducible**: Consistent test environment across machines and CI/CD
- **Real protocols**: Catches issues that mocks can't detect

#### CI/CD Integration

Integration tests can run in GitHub Actions or other CI systems with Docker support:

```yaml
integration-tests:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
    - name: Run integration tests
      run: make test-int
```

#### Future Integration Tests

**Planned:**
- **MQTT**: Mosquitto container for MQTT publisher testing
- **Solace**: Official Solace PubSub+ container (supports both MQTT and SMF protocols)
- **Epever End-to-End**: Full collection → publish → verify flow using built-in Modbus simulator

**Design Principles:**
1. Use build tags to separate integration from unit tests
2. One integration test file per publisher/component
3. Reusable container helpers in `internal/testing/containers/`
4. Parallel test support via dynamic port mapping
5. Comprehensive cleanup with `t.Cleanup()`

## Important Notes

- The React app must be built before building the Go binary (since it's embedded via `//go:embed`)
- The build process copies `site/build` to `internal/static/build` where it's embedded into the binary
- Main package is in `cmd/controller/`, not at the project root
- Controllers implement graceful shutdown via `defer controller.Close()` in `main.go`
- Message publishing is optional - if no publisher (MQTT, Solace, File, or RemoteWrite) is enabled, a no-op publisher is used
- Publishers implement the `MessagePublisher` interface for easy testing and swapping
- Multiple message publishers can be enabled simultaneously - metrics are published to all enabled publishers
- The application uses structured logging via `logrus`
- All controllers register their own HTTP endpoints via `RegisterEndpoints()`
- Always add unit tests for new code
- Always run linters after code changes