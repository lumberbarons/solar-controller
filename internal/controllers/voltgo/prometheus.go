package voltgo

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// batteryLabel names the series dimension that separates one battery from
// another. Every voltgo series carries it, including the single-battery case,
// so a query written against one battery keeps working when a second is added.
const batteryLabel = "battery"

// PrometheusCollector exposes every configured battery through one set of
// labelled metrics. One instance is shared by all batteries: promauto
// registers into the default registry, which panics on a duplicate
// registration, so a per-battery collector is not an option.
type PrometheusCollector struct {
	failures *prometheus.CounterVec

	batteryVoltage *prometheus.GaugeVec
	batteryCurrent *prometheus.GaugeVec
	batteryPower   *prometheus.GaugeVec
	batterySoc     *prometheus.GaugeVec
	batterySoh     *prometheus.GaugeVec
	batteryTemp    *prometheus.GaugeVec

	cellVoltageDelta *prometheus.GaugeVec
	cellVoltage      *prometheus.GaugeVec
}

func NewPrometheusCollector() *PrometheusCollector {
	endpoint := &PrometheusCollector{
		failures: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "read_failures",
			Help:      "Number of errors while reading from the voltgo battery.",
		}, []string{batteryLabel}),
	}

	// Initialize all metrics immediately to avoid race conditions
	endpoint.initializeMetrics()

	return endpoint
}

func (v *PrometheusCollector) IncrementFailures(batteryID string) {
	v.failures.WithLabelValues(batteryID).Inc()
}

func (v *PrometheusCollector) initializeMetrics() {
	v.batteryVoltage = v.newBatteryGauge("battery_voltage", "Battery pack voltage (V).")
	v.batteryCurrent = v.newBatteryGauge("battery_current",
		"Battery pack current (A), positive when charging, negative when discharging.")
	v.batteryPower = v.newBatteryGauge("battery_power",
		"Battery pack power (W), derived from voltage and current.")
	v.batterySoc = v.newBatteryGauge("battery_soc", "Battery state of charge (%).")
	v.batterySoh = v.newBatteryGauge("battery_soh", "Battery state of health (%).")
	v.batteryTemp = v.newBatteryGauge("battery_temp", "Battery temperature (C).")
	v.cellVoltageDelta = v.newBatteryGauge("cell_voltage_delta",
		"Spread between the highest and lowest cell voltage (V).")

	v.cellVoltage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "cell_voltage",
			Help:      "Individual cell voltage (V).",
		},
		[]string{batteryLabel, "cell"},
	)
}

func (v *PrometheusCollector) newBatteryGauge(name, help string) *prometheus.GaugeVec {
	return promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      name,
			Help:      help,
		},
		[]string{batteryLabel},
	)
}

func (v *PrometheusCollector) SetMetrics(batteryID string, status *BatteryStatus) {
	v.batteryVoltage.WithLabelValues(batteryID).Set(status.Voltage)
	v.batteryCurrent.WithLabelValues(batteryID).Set(status.Current)
	v.batteryPower.WithLabelValues(batteryID).Set(status.Voltage * status.Current)
	v.batterySoc.WithLabelValues(batteryID).Set(float64(status.SOC))
	v.batterySoh.WithLabelValues(batteryID).Set(float64(status.SOH))
	v.batteryTemp.WithLabelValues(batteryID).Set(status.Temperature)

	v.cellVoltageDelta.WithLabelValues(batteryID).Set(cellVoltageDelta(status.Cells))
	for _, cell := range status.Cells {
		v.cellVoltage.WithLabelValues(batteryID, strconv.Itoa(cell.Index)).Set(cell.Voltage)
	}
}
