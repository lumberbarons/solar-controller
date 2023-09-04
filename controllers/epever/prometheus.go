package epever

import (
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

const (
	namespace = "epever"
)

type PrometheusCollector struct {
	epeverCollector *Collector

	scrapeFailures prometheus.Counter

	panelVoltage *prometheus.Desc
	panelCurrent *prometheus.Desc
	panelPower   *prometheus.Desc

	chargingPower   *prometheus.Desc
	chargingCurrent *prometheus.Desc

	batteryVoltage    *prometheus.Desc
	batterySOC        *prometheus.Desc
	batteryTemp       *prometheus.Desc
	batteryMaxVoltage *prometheus.Desc
	batteryMinVoltage *prometheus.Desc

	deviceTemp *prometheus.Desc

	energyGeneratedDaily   *prometheus.Desc
	energyGeneratedMonthly *prometheus.Desc

	chargingStatus *prometheus.Desc
}

func NewPrometheusCollector(collector *Collector) *PrometheusCollector {
	endpoint := &PrometheusCollector{
		epeverCollector: collector,

		scrapeFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "controller_comm_failures_total",
			Help:      "Number of communications errors while connecting to the solar controller.",
		}),
		panelVoltage: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "panel_voltage"),
			"Solar panel voltage (V).",
			nil,
			nil,
		),
		panelCurrent: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "panel_current"),
			"Solar panel current (A).",
			nil,
			nil,
		),
		panelPower: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "panel_power"),
			"Solar panel power (W).",
			nil,
			nil,
		),
		chargingPower: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "charging_power"),
			"Battery charging power (W).",
			nil,
			nil,
		),
		chargingCurrent: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "charging_current"),
			"Battery charging current (A).",
			nil,
			nil,
		),
		chargingStatus: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "charging_status"),
			"Charging status.",
			nil,
			nil,
		),
		batteryVoltage: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "battery_voltage"),
			"Battery voltage (V).",
			nil,
			nil,
		),
		batterySOC: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "battery_soc"),
			"Battery state of charge (%).",
			nil,
			nil,
		),
		batteryTemp: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "battery_temp"),
			"Battery temperature (C).",
			nil,
			nil,
		),
		batteryMaxVoltage: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "battery_max_voltage"),
			"Maximum battery voltage (V).",
			nil,
			nil,
		),
		batteryMinVoltage: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "battery_min_voltage"),
			"Minimum battery voltage (V).",
			nil,
			nil,
		),

		deviceTemp: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "device_temp"),
			"Controller temperature (C).",
			nil,
			nil,
		),

		energyGeneratedDaily: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "energy_generated_daily"),
			"Controller calculated daily power generation, (kWh).",
			nil,
			nil,
		),
		energyGeneratedMonthly: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "energy_generated_monthly"),
			"Controller calculated monthly power generation, (kWh).",
			nil,
			nil,
		),
	}

	return endpoint
}

func (e *PrometheusCollector) Describe(ch chan <- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		e.panelVoltage,
		e.panelCurrent,
		e.panelPower,
		e.chargingPower,
		e.chargingCurrent,
		e.batteryVoltage,
		e.batterySOC,
		e.batteryTemp,
		e.batteryMaxVoltage,
		e.batteryMinVoltage,
		e.deviceTemp,
		e.energyGeneratedDaily,
		e.energyGeneratedMonthly,
		e.chargingStatus,
	}

	for _, d := range ds {
		ch <- d
	}
}

func (e *PrometheusCollector) Collect(ch chan <- prometheus.Metric) {
	if err := e.collect(ch); err != nil {
		log.Printf("Error getting solar controller data: %s", err)
		e.scrapeFailures.Inc()
		e.scrapeFailures.Collect(ch)
	}

	return
}

func (e *PrometheusCollector) collect(ch chan <- prometheus.Metric) error {
	status, err :=  e.epeverCollector.GetStatus()
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(
		e.panelVoltage,
		prometheus.GaugeValue,
		float64(status.ArrayVoltage),
	)
	ch <- prometheus.MustNewConstMetric(
		e.panelCurrent,
		prometheus.GaugeValue,
		float64(status.ArrayCurrent),
	)
	ch <- prometheus.MustNewConstMetric(
		e.panelPower,
		prometheus.GaugeValue,
		float64(status.ArrayPower),
	)

	ch <- prometheus.MustNewConstMetric(
		e.chargingPower,
		prometheus.GaugeValue,
		float64(status.ChargingPower),
	)
	ch <- prometheus.MustNewConstMetric(
		e.chargingCurrent,
		prometheus.GaugeValue,
		float64(status.ChargingCurrent),
	)

	ch <- prometheus.MustNewConstMetric(
		e.batteryVoltage,
		prometheus.GaugeValue,
		float64(status.BatteryVoltage),
	)
	ch <- prometheus.MustNewConstMetric(
		e.batterySOC,
		prometheus.GaugeValue,
		float64(status.BatterySOC),
	)
	ch <- prometheus.MustNewConstMetric(
		e.batteryTemp,
		prometheus.GaugeValue,
		float64(status.BatteryTemp),
	)
	ch <- prometheus.MustNewConstMetric(
		e.batteryMinVoltage,
		prometheus.GaugeValue,
		float64(status.BatteryMinVoltage),
	)
	ch <- prometheus.MustNewConstMetric(
		e.batteryMaxVoltage,
		prometheus.GaugeValue,
		float64(status.BatteryMaxVoltage),
	)

	ch <- prometheus.MustNewConstMetric(
		e.deviceTemp,
		prometheus.GaugeValue,
		float64(status.DeviceTemp),
	)

	ch <- prometheus.MustNewConstMetric(
		e.energyGeneratedDaily,
		prometheus.GaugeValue,
		float64(status.EnergyGeneratedDaily),
	)

	ch <- prometheus.MustNewConstMetric(
		e.chargingStatus,
		prometheus.GaugeValue,
		float64(status.ChargingStatus),
	)

	return nil
}
