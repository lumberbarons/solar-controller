package pijuice

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type PrometheusCollector struct {
	failures prometheus.Counter

	batteryStatus    prometheus.Gauge
	powerInputStatus prometheus.Gauge

	batteryCharge  prometheus.Gauge
	batteryTemp    prometheus.Gauge
	batteryVoltage prometheus.Gauge
	batteryCurrent prometheus.Gauge
}

func NewPrometheusCollector() *PrometheusCollector {
	endpoint := &PrometheusCollector{
		failures: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "failures",
			Help:      "Number of errors while connecting to the pijuice hat.",
		}),
	}

	return endpoint
}

func (e *PrometheusCollector) initializeMetrics() {
	e.batteryStatus = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "battery_status",
		Help:      "Battery status.",
	})

	e.powerInputStatus = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "power_input_status",
		Help:      "Power input status.",
	})

	e.batteryCharge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "battery_chargge",
		Help:      "Battery charge (%).",
	})

	e.batteryTemp = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "battery_temp",
		Help:      "Battery temperature (C)",
	})

	e.batteryVoltage = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "battery_voltage",
		Help:      "Battery Voltage (V).",
	})

	e.batteryCurrent = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "battery_current",
		Help:      "Battery current (A).",
	})
}

func (e *PrometheusCollector) IncrementFailures() {
	e.failures.Inc()
}

func (e *PrometheusCollector) SetMetrics(status *ControllerStatus) {
	if e.batteryStatus == nil {
		e.initializeMetrics()
	}

	e.batteryStatus.Set(float64(status.BatteryStatus))
	e.powerInputStatus.Set(float64(status.PowerInputStatus))
}