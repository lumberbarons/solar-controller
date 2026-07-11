package voltgo

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type PrometheusCollector struct {
	failures prometheus.Counter

	batteryVoltage prometheus.Gauge
	batteryCurrent prometheus.Gauge
	batteryPower   prometheus.Gauge
	batterySoc     prometheus.Gauge
	batterySoh     prometheus.Gauge
	batteryTemp    prometheus.Gauge

	cellVoltageDelta prometheus.Gauge
	cellVoltage      *prometheus.GaugeVec
}

func NewPrometheusCollector() *PrometheusCollector {
	endpoint := &PrometheusCollector{
		failures: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "read_failures",
			Help:      "Number of errors while reading from the voltgo battery.",
		}),
	}

	// Initialize all metrics immediately to avoid race conditions
	endpoint.initializeMetrics()

	return endpoint
}

func (v *PrometheusCollector) IncrementFailures() {
	v.failures.Inc()
}

func (v *PrometheusCollector) initializeMetrics() {
	v.batteryVoltage = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "battery_voltage",
		Help:      "Battery pack voltage (V).",
	})

	v.batteryCurrent = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "battery_current",
		Help:      "Battery pack current (A), positive when charging, negative when discharging.",
	})

	v.batteryPower = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "battery_power",
		Help:      "Battery pack power (W), derived from voltage and current.",
	})

	v.batterySoc = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "battery_soc",
		Help:      "Battery state of charge (%).",
	})

	v.batterySoh = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "battery_soh",
		Help:      "Battery state of health (%).",
	})

	v.batteryTemp = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "battery_temp",
		Help:      "Battery temperature (C).",
	})

	v.cellVoltageDelta = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "cell_voltage_delta",
		Help:      "Spread between the highest and lowest cell voltage (V).",
	})

	v.cellVoltage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "cell_voltage",
			Help:      "Individual cell voltage (V).",
		},
		[]string{"cell"},
	)
}

func (v *PrometheusCollector) SetMetrics(status *BatteryStatus) {
	v.batteryVoltage.Set(status.Voltage)
	v.batteryCurrent.Set(status.Current)
	v.batteryPower.Set(status.Voltage * status.Current)
	v.batterySoc.Set(float64(status.SOC))
	v.batterySoh.Set(float64(status.SOH))
	v.batteryTemp.Set(status.Temperature)

	v.cellVoltageDelta.Set(cellVoltageDelta(status.Cells))
	for _, cell := range status.Cells {
		v.cellVoltage.WithLabelValues(strconv.Itoa(cell.Index)).Set(cell.Voltage)
	}
}
