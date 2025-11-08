package epever

import (
	"encoding/json"
	"fmt"
)

// Metric represents a single metric with its value, unit, and timestamp
type Metric struct {
	Name      string
	Value     interface{}
	Unit      string
	Timestamp int64
}

// MetricPayload is the JSON structure published for each metric
type MetricPayload struct {
	Value     interface{} `json:"value"`
	Unit      string      `json:"unit"`
	Timestamp int64       `json:"timestamp"`
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

// ConvertStatusToMetrics converts a ControllerStatus into individual metrics
func ConvertStatusToMetrics(status *ControllerStatus) []Metric {
	if status == nil {
		return []Metric{}
	}

	timestamp := status.Timestamp

	return []Metric{
		{
			Name:      "array-voltage",
			Value:     status.ArrayVoltage,
			Unit:      "volts",
			Timestamp: timestamp,
		},
		{
			Name:      "array-current",
			Value:     status.ArrayCurrent,
			Unit:      "amperes",
			Timestamp: timestamp,
		},
		{
			Name:      "array-power",
			Value:     status.ArrayPower,
			Unit:      "watts",
			Timestamp: timestamp,
		},
		{
			Name:      "charging-current",
			Value:     status.ChargingCurrent,
			Unit:      "amperes",
			Timestamp: timestamp,
		},
		{
			Name:      "charging-power",
			Value:     status.ChargingPower,
			Unit:      "watts",
			Timestamp: timestamp,
		},
		{
			Name:      "battery-voltage",
			Value:     status.BatteryVoltage,
			Unit:      "volts",
			Timestamp: timestamp,
		},
		{
			Name:      "battery-soc",
			Value:     status.BatterySOC,
			Unit:      "percent",
			Timestamp: timestamp,
		},
		{
			Name:      "battery-temp",
			Value:     status.BatteryTemp,
			Unit:      "celsius",
			Timestamp: timestamp,
		},
		{
			Name:      "device-temp",
			Value:     status.DeviceTemp,
			Unit:      "celsius",
			Timestamp: timestamp,
		},
		{
			Name:      "energy-generated-daily",
			Value:     status.EnergyGeneratedDaily,
			Unit:      "kilowatt-hours",
			Timestamp: timestamp,
		},
		{
			Name:      "charging-status",
			Value:     status.ChargingStatus,
			Unit:      "code",
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
