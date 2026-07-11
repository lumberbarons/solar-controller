package voltgo

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

// Verify PrometheusCollector implements MetricsCollector
var _ MetricsCollector = (*PrometheusCollector)(nil)

// The collector registers with the global Prometheus registry via promauto,
// so it can only be created once per test binary.
func TestPrometheusCollector(t *testing.T) {
	collector := NewPrometheusCollector()

	t.Run("SetMetrics updates all gauges", func(t *testing.T) {
		status := &BatteryStatus{
			Voltage:     13.28,
			Current:     -2.5,
			SOC:         87,
			SOH:         100,
			Temperature: 21.5,
			CellCount:   4,
			Cells: []Cell{
				{Index: 0, Voltage: 3.321},
				{Index: 1, Voltage: 3.318},
				{Index: 2, Voltage: 3.322},
				{Index: 3, Voltage: 3.319},
			},
		}

		collector.SetMetrics(status)

		checks := []struct {
			name string
			got  float64
			want float64
		}{
			{"battery_voltage", testutil.ToFloat64(collector.batteryVoltage), 13.28},
			{"battery_current", testutil.ToFloat64(collector.batteryCurrent), -2.5},
			{"battery_power", testutil.ToFloat64(collector.batteryPower), status.Voltage * status.Current},
			{"battery_soc", testutil.ToFloat64(collector.batterySoc), 87},
			{"battery_soh", testutil.ToFloat64(collector.batterySoh), 100},
			{"battery_temp", testutil.ToFloat64(collector.batteryTemp), 21.5},
			{"cell_voltage_delta", testutil.ToFloat64(collector.cellVoltageDelta), status.Cells[2].Voltage - status.Cells[1].Voltage},
		}

		for _, c := range checks {
			if c.got != c.want {
				t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
			}
		}

		if got := testutil.ToFloat64(collector.cellVoltage.WithLabelValues("2")); got != 3.322 {
			t.Errorf("cell_voltage{cell=\"2\"} = %v, want 3.322", got)
		}
	})

	t.Run("IncrementFailures increments the counter", func(t *testing.T) {
		before := testutil.ToFloat64(collector.failures)
		collector.IncrementFailures()
		if got := testutil.ToFloat64(collector.failures); got != before+1 {
			t.Errorf("read_failures = %v, want %v", got, before+1)
		}
	})
}
