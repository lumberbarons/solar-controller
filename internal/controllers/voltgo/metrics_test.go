package voltgo

import (
	"encoding/json"
	"testing"
	"time"
)

func TestConvertStatusToMetrics(t *testing.T) {
	t.Run("converts all metrics with correct names, units, and values", func(t *testing.T) {
		status := &BatteryStatus{
			Timestamp:      1699000000,
			CollectionTime: 0.42,
			Voltage:        13.28,
			Current:        -2.5,
			SOC:            87,
			SOH:            100,
			Temperature:    21.5,
			Temperatures:   []int{21, 22},
			CellCount:      4,
			Cells: []Cell{
				{Index: 0, Voltage: 3.321},
				{Index: 1, Voltage: 3.318},
				{Index: 2, Voltage: 3.322},
				{Index: 3, Voltage: 3.319},
			},
		}

		metrics := ConvertStatusToMetrics(status)

		want := []struct {
			name  string
			value any
			unit  string
		}{
			{"battery-voltage", 13.28, "volts"},
			{"battery-current", -2.5, "amperes"},
			{"battery-power", status.Voltage * status.Current, "watts"},
			{"battery-soc", 87, "percent"},
			{"battery-soh", 100, "percent"},
			{"battery-temp", 21.5, "celsius"},
			{"cell-voltage-delta", status.Cells[2].Voltage - status.Cells[1].Voltage, "volts"},
			{"collection-time", 0.42, "seconds"},
		}

		if len(metrics) != len(want) {
			t.Fatalf("metrics length = %d, want %d", len(metrics), len(want))
		}

		for i, w := range want {
			m := metrics[i]
			if m.Name != w.name {
				t.Errorf("metrics[%d].Name = %s, want %s", i, m.Name, w.name)
			}
			if m.Unit != w.unit {
				t.Errorf("%s Unit = %s, want %s", w.name, m.Unit, w.unit)
			}
			if m.Value != w.value {
				t.Errorf("%s Value = %v, want %v", w.name, m.Value, w.value)
			}
			if m.Timestamp != status.Timestamp {
				t.Errorf("%s Timestamp = %d, want %d", w.name, m.Timestamp, status.Timestamp)
			}
		}
	})

	t.Run("nil status returns empty slice", func(t *testing.T) {
		metrics := ConvertStatusToMetrics(nil)
		if len(metrics) != 0 {
			t.Errorf("metrics length = %d, want 0", len(metrics))
		}
	})

	t.Run("no cells yields zero voltage delta", func(t *testing.T) {
		metrics := ConvertStatusToMetrics(&BatteryStatus{})

		for _, m := range metrics {
			if m.Name == "cell-voltage-delta" {
				if m.Value != 0.0 {
					t.Errorf("cell-voltage-delta = %v, want 0", m.Value)
				}
				return
			}
		}
		t.Fatal("cell-voltage-delta metric not found")
	})
}

func TestMetricToJSON(t *testing.T) {
	metric := Metric{
		Name:      "battery-voltage",
		Value:     13.28,
		Unit:      "volts",
		Timestamp: 1699000000,
	}

	jsonStr, err := metric.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &payload); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if payload["value"] != 13.28 {
		t.Errorf("value = %v, want 13.28", payload["value"])
	}
	if payload["unit"] != "volts" {
		t.Errorf("unit = %v, want volts", payload["unit"])
	}
	if payload["timestamp"] != float64(1699000000) {
		t.Errorf("timestamp = %v, want 1699000000", payload["timestamp"])
	}
}

func TestCreateCollectionFailureMetric(t *testing.T) {
	before := time.Now().Unix()
	metric := CreateCollectionFailureMetric()
	after := time.Now().Unix()

	if metric.Name != "collection-failure" {
		t.Errorf("Name = %s, want collection-failure", metric.Name)
	}
	if metric.Value != 1 {
		t.Errorf("Value = %v, want 1", metric.Value)
	}
	if metric.Unit != "count" {
		t.Errorf("Unit = %s, want count", metric.Unit)
	}
	if metric.Timestamp < before || metric.Timestamp > after {
		t.Errorf("Timestamp = %d, want between %d and %d", metric.Timestamp, before, after)
	}
}
