package victron

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type PrometheusCollector struct {
	failures prometheus.Counter

	consumedAh    prometheus.Gauge
	power         prometheus.Gauge
	voltage       prometheus.Gauge
	current       prometheus.Gauge
	stateOfCharge prometheus.Gauge
}

func NewPrometheusCollector() *PrometheusCollector {
	endpoint := &PrometheusCollector{
		failures: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "failures",
			Help:      "Number of errors while connecting to the victron controller.",
		}),

		consumedAh: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "consumed_ah",
			Help:      "Consumed capacity (Ah).",
		}),

		power: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "power",
			Help:      "Power (W).",
		}),

		voltage: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "voltage",
			Help:      "Voltage (V).",
		}),

		current: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "curent",
			Help:      "Current (A).",
		}),

		stateOfCharge: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "state_of_charge",
			Help:      "State of charge (%).",
		}),
	}

	return endpoint
}

func (e *PrometheusCollector) IncrementFailures() {
	e.failures.Inc()
}

func (e *PrometheusCollector) SetMetrics(status *ControllerStatus) {
	e.consumedAh.Set(float64(status.ConsumedAh))
	e.power.Set(float64(status.Power))
	e.voltage.Set(float64(status.Voltage))
	e.current.Set(float64(status.Current))
	e.stateOfCharge.Set(float64(status.StateOfCharge))
}
