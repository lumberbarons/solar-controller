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
	}

	return endpoint
}

func (e *PrometheusCollector) initializeMetrics() {
	e.consumedAh = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "consumed_ah",
		Help:      "Consumed capacity (Ah).",
	})

	e.power = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "power",
		Help:      "Power (W).",
	})

	e.voltage = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "voltage",
		Help:      "Voltage (V).",
	})

	e.current = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "current",
		Help:      "Current (A).",
	})

	e.stateOfCharge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "state_of_charge",
		Help:      "State of charge (%).",
	})
}

func (e *PrometheusCollector) IncrementFailures() {
	e.failures.Inc()
}

func (e *PrometheusCollector) SetMetrics(status *ControllerStatus) {
	if e.consumedAh == nil {
		e.initializeMetrics()
	}

	e.consumedAh.Set(float64(status.ConsumedAh))
	e.power.Set(float64(status.Power))
	e.voltage.Set(float64(status.Voltage))
	e.current.Set(float64(status.Current))
	e.stateOfCharge.Set(float64(status.StateOfCharge))
}
