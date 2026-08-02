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

	statusFor := func(voltage float64, soc int) *BatteryStatus {
		return &BatteryStatus{
			Voltage:     voltage,
			Current:     -2.5,
			SOC:         soc,
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
	}

	t.Run("SetMetrics updates all gauges under the battery label", func(t *testing.T) {
		status := statusFor(13.28, 87)
		collector.SetMetrics("bank-a", status)

		checks := []struct {
			name string
			got  float64
			want float64
		}{
			{"battery_voltage", testutil.ToFloat64(collector.batteryVoltage.WithLabelValues("bank-a")), 13.28},
			{"battery_current", testutil.ToFloat64(collector.batteryCurrent.WithLabelValues("bank-a")), -2.5},
			{"battery_power", testutil.ToFloat64(collector.batteryPower.WithLabelValues("bank-a")), status.Voltage * status.Current},
			{"battery_soc", testutil.ToFloat64(collector.batterySoc.WithLabelValues("bank-a")), 87},
			{"battery_soh", testutil.ToFloat64(collector.batterySoh.WithLabelValues("bank-a")), 100},
			{"battery_temp", testutil.ToFloat64(collector.batteryTemp.WithLabelValues("bank-a")), 21.5},
			{"cell_voltage_delta", testutil.ToFloat64(collector.cellVoltageDelta.WithLabelValues("bank-a")), status.Cells[2].Voltage - status.Cells[1].Voltage},
		}

		for _, c := range checks {
			if c.got != c.want {
				t.Errorf("%s{battery=\"bank-a\"} = %v, want %v", c.name, c.got, c.want)
			}
		}

		if got := testutil.ToFloat64(collector.cellVoltage.WithLabelValues("bank-a", "2")); got != 3.322 {
			t.Errorf("cell_voltage{battery=\"bank-a\",cell=\"2\"} = %v, want 3.322", got)
		}
	})

	// Without the battery label, four packs would overwrite each other's
	// values and the dashboard would show whichever collected last.
	t.Run("batteries do not overwrite each other", func(t *testing.T) {
		collector.SetMetrics("bank-a", statusFor(13.28, 87))
		collector.SetMetrics("bank-b", statusFor(12.90, 64))

		if got := testutil.ToFloat64(collector.batteryVoltage.WithLabelValues("bank-a")); got != 13.28 {
			t.Errorf("battery_voltage{battery=\"bank-a\"} = %v, want 13.28", got)
		}
		if got := testutil.ToFloat64(collector.batteryVoltage.WithLabelValues("bank-b")); got != 12.90 {
			t.Errorf("battery_voltage{battery=\"bank-b\"} = %v, want 12.90", got)
		}
		if got := testutil.ToFloat64(collector.batterySoc.WithLabelValues("bank-a")); got != 87 {
			t.Errorf("battery_soc{battery=\"bank-a\"} = %v, want 87", got)
		}
		if got := testutil.ToFloat64(collector.batterySoc.WithLabelValues("bank-b")); got != 64 {
			t.Errorf("battery_soc{battery=\"bank-b\"} = %v, want 64", got)
		}
	})

	// The cell label alone is not enough: cell 2 of one pack and cell 2 of
	// another are different cells.
	t.Run("cell voltages are separated by battery as well as cell", func(t *testing.T) {
		perBattery := statusFor(13.28, 87)
		perBattery.Cells = []Cell{{Index: 0, Voltage: 3.400}}
		collector.SetMetrics("bank-c", perBattery)

		if got := testutil.ToFloat64(collector.cellVoltage.WithLabelValues("bank-c", "0")); got != 3.400 {
			t.Errorf("cell_voltage{battery=\"bank-c\",cell=\"0\"} = %v, want 3.400", got)
		}
		if got := testutil.ToFloat64(collector.cellVoltage.WithLabelValues("bank-a", "0")); got != 3.321 {
			t.Errorf("cell_voltage{battery=\"bank-a\",cell=\"0\"} = %v, want 3.321 (bank-c must not overwrite it)", got)
		}
	})

	t.Run("IncrementFailures increments the counter for one battery only", func(t *testing.T) {
		beforeA := testutil.ToFloat64(collector.failures.WithLabelValues("bank-a"))
		beforeB := testutil.ToFloat64(collector.failures.WithLabelValues("bank-b"))

		collector.IncrementFailures("bank-a")

		if got := testutil.ToFloat64(collector.failures.WithLabelValues("bank-a")); got != beforeA+1 {
			t.Errorf("read_failures{battery=\"bank-a\"} = %v, want %v", got, beforeA+1)
		}
		if got := testutil.ToFloat64(collector.failures.WithLabelValues("bank-b")); got != beforeB {
			t.Errorf("read_failures{battery=\"bank-b\"} = %v, want %v (unchanged)", got, beforeB)
		}
	})
}
