package voltgo

import (
	"encoding/json"
	"fmt"
	"time"
)

// Metric represents a single metric with its value, unit, and timestamp
type Metric struct {
	Name      string
	Value     any
	Unit      string
	Timestamp int64
}

// MetricPayload is the JSON structure published for each metric
type MetricPayload struct {
	Value     any    `json:"value"`
	Unit      string `json:"unit"`
	Timestamp int64  `json:"timestamp"`
}

// ToJSON converts a Metric to its JSON representation
func (m *Metric) ToJSON() (string, error) {
	payload := MetricPayload{
		Value:     m.Value,
		Unit:      m.Unit,
		Timestamp: m.Timestamp,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metric payload: %w", err)
	}

	return string(b), nil
}

// ConvertStatusToMetrics converts a BatteryStatus into individual metrics.
// Per-cell voltages are deliberately not converted - they are exposed via
// Prometheus and the HTTP API only, to keep the published topic space bounded.
func ConvertStatusToMetrics(status *BatteryStatus) []Metric {
	if status == nil {
		return []Metric{}
	}

	timestamp := status.Timestamp

	return []Metric{
		{
			Name:      "battery-voltage",
			Value:     status.Voltage,
			Unit:      "volts",
			Timestamp: timestamp,
		},
		{
			Name:      "battery-current",
			Value:     status.Current,
			Unit:      "amperes",
			Timestamp: timestamp,
		},
		{
			Name:      "battery-power",
			Value:     status.Voltage * status.Current,
			Unit:      "watts",
			Timestamp: timestamp,
		},
		{
			Name:      "battery-soc",
			Value:     status.SOC,
			Unit:      "percent",
			Timestamp: timestamp,
		},
		{
			Name:      "battery-soh",
			Value:     status.SOH,
			Unit:      "percent",
			Timestamp: timestamp,
		},
		{
			Name:      "battery-temp",
			Value:     status.Temperature,
			Unit:      "celsius",
			Timestamp: timestamp,
		},
		{
			Name:      "cell-voltage-delta",
			Value:     cellVoltageDelta(status.Cells),
			Unit:      "volts",
			Timestamp: timestamp,
		},
		{
			Name:      "collection-time",
			Value:     status.CollectionTime,
			Unit:      "seconds",
			Timestamp: timestamp,
		},
	}
}

// cellVoltageDelta returns the spread between the highest and lowest cell
// voltage, or 0 when there are no cells.
func cellVoltageDelta(cells []Cell) float64 {
	if len(cells) == 0 {
		return 0
	}

	minV, maxV := cells[0].Voltage, cells[0].Voltage
	for _, cell := range cells[1:] {
		if cell.Voltage < minV {
			minV = cell.Voltage
		}
		if cell.Voltage > maxV {
			maxV = cell.Voltage
		}
	}
	return maxV - minV
}

// CreateCollectionFailureMetric creates a failure metric when collection fails
func CreateCollectionFailureMetric() Metric {
	return Metric{
		Name:      "collection-failure",
		Value:     1,
		Unit:      "count",
		Timestamp: time.Now().Unix(),
	}
}
