package epever

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type PrometheusCollector struct {
	failures      prometheus.Counter
	writeFailures prometheus.Counter

	panelVoltage prometheus.Gauge
	panelCurrent prometheus.Gauge
	panelPower   prometheus.Gauge

	chargingPower   prometheus.Gauge
	chargingCurrent prometheus.Gauge

	batteryVoltage    prometheus.Gauge
	batterySoc        prometheus.Gauge
	batteryTemp       prometheus.Gauge
	batteryMinVoltage prometheus.Gauge
	batteryMaxVoltage prometheus.Gauge

	deviceTemp prometheus.Gauge

	energyGeneratedDaily prometheus.Gauge

	chargingStatus prometheus.Gauge
}

func NewPrometheusCollector() *PrometheusCollector {
	endpoint := &PrometheusCollector{
		failures: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "read_failures",
			Help:      "Number of errors while reading from the epever controller.",
		}),
		writeFailures: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "write_failures",
			Help:      "Number of errors while writing to the epever controller.",
		}),
	}

	// Initialize all metrics immediately to avoid race conditions
	endpoint.initializeMetrics()

	return endpoint
}

func (e *PrometheusCollector) IncrementFailures() {
	e.failures.Inc()
}

func (e *PrometheusCollector) IncrementWriteFailures() {
	e.writeFailures.Inc()
}

func (e *PrometheusCollector) initializeMetrics() {
	e.panelVoltage = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "panel_voltage",
		Help:      "Solar panel voltage (V).",
	})

	e.panelCurrent = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "panel_current",
		Help:      "Solar panel current (A).",
	})

	e.panelPower = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "panel_power",
		Help:      "Solar panel power (W).",
	})

	e.chargingPower = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "charging_power",
		Help:      "Battery charging power (W).",
	})

	e.chargingCurrent = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "charging_current",
		Help:      "Battery charging current (A).",
	})

	e.batteryVoltage = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "battery_voltage",
		Help:      "Battery voltage (V).",
	})

	e.batterySoc = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "battery_soc",
		Help:      "Battery state of charge (%).",
	})

	e.batteryTemp = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "battery_temp",
		Help:      "Battery temperature (C).",
	})

	e.batteryMinVoltage = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "battery_min_voltage",
		Help:      "Minimum battery voltage (V).",
	})

	e.batteryMaxVoltage = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "battery_max_voltage",
		Help:      "Maximum battery voltage (V).",
	})

	e.deviceTemp = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "device_temp",
		Help:      "Controller temperature (C).",
	})

	e.energyGeneratedDaily = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "energy_generated_daily",
		Help:      "Controller calculated daily power generation, (kWh).",
	})

	e.chargingStatus = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "charging_status",
		Help:      "Charging status.",
	})
}

func (e *PrometheusCollector) SetMetrics(status *ControllerStatus) {
	e.panelVoltage.Set(float64(status.ArrayVoltage))
	e.panelCurrent.Set(float64(status.ArrayCurrent))
	e.panelPower.Set(float64(status.ArrayPower))

	e.chargingPower.Set(float64(status.ChargingPower))
	e.chargingCurrent.Set(float64(status.ChargingCurrent))

	e.batteryVoltage.Set(float64(status.BatteryVoltage))
	e.batterySoc.Set(float64(status.BatterySOC))
	e.batteryTemp.Set(float64(status.BatteryTemp))
	e.batteryMinVoltage.Set(float64(status.BatteryMinVoltage))
	e.batteryMaxVoltage.Set(float64(status.BatteryMaxVoltage))

	e.deviceTemp.Set(float64(status.DeviceTemp))

	e.energyGeneratedDaily.Set(float64(status.EnergyGeneratedDaily))

	e.chargingStatus.Set(float64(status.ChargingStatus))
}
