package voltgo

import (
	"context"

	"github.com/lumberbarons/voltgo/battery"
)

// BatteryClient defines the interface for a connected voltgo battery.
// This abstraction allows for testing without physical hardware.
type BatteryClient interface {
	// GetStatus reads the full battery status (voltage, current, SOC, SOH,
	// temperatures, and per-cell voltages) in a single call.
	GetStatus(ctx context.Context) (*battery.Status, error)

	// GetInfo reads static battery information (chemistry, nominal voltage,
	// capacity, device identity strings).
	GetInfo(ctx context.Context) (*battery.Info, error)

	// IsConnected reports whether the BLE connection is still alive.
	IsConnected() bool

	// Disconnect closes the BLE connection to the battery.
	Disconnect() error
}

// MetricsCollector defines the interface for collecting and exposing metrics.
// This abstraction allows for testing without the Prometheus global registry.
type MetricsCollector interface {
	// IncrementFailures increments the collection failure counter.
	IncrementFailures()

	// SetMetrics updates all metrics based on the provided status.
	SetMetrics(status *BatteryStatus)
}

// BatteryConnector defines the interface for establishing BLE connections
// to voltgo batteries.
type BatteryConnector interface {
	// Connect establishes a BLE connection to the battery at the given address.
	Connect(ctx context.Context, address string) (BatteryClient, error)

	// Close releases the underlying BLE adapter.
	Close() error
}
