package exporter

import (
	"github.com/lumberbarons/solar-controller/epever/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	namespace = "solar"
)

type PrometheusEndpoint struct {
	solarCollector *collector.EpeverCollector

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

	Handler http.Handler
}

func NewPrometheusEndpoint(collector *collector.EpeverCollector) *PrometheusEndpoint {
	prometheus.MustRegister()

	endpoint := &PrometheusEndpoint{
		solarCollector: collector,

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

	registry := prometheus.NewRegistry()
	registry.MustRegister(endpoint)
	endpoint.Handler = promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	return endpoint
}

func (pe *PrometheusEndpoint) Describe(ch chan <- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		pe.panelVoltage,
	}

	for _, d := range ds {
		ch <- d
	}

	pe.scrapeFailures.Describe(ch)
}

func (pe *PrometheusEndpoint) Collect(ch chan<- prometheus.Metric) {
	if err := pe.collect(ch); err != nil {
		log.Printf("Error getting solar controller data: %s", err)
		pe.scrapeFailures.Inc()
		pe.scrapeFailures.Collect(ch)
	}

	return
}

func start(s string) (string, time.Time) {
	log.Printf("start %s", s)
	return s, time.Now()
}


func end(s string, startTime time.Time) {
	endTime := time.Now()
	log.Printf("end %s, time: %.4f sec", s, endTime.Sub(startTime).Seconds())
}

func (pe *PrometheusEndpoint) collect(ch chan <- prometheus.Metric) error {
	defer end(start("collector collection"))

	status, err :=  pe.solarCollector.GetStatus()
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(
		pe.panelVoltage,
		prometheus.GaugeValue,
		float64(status.ArrayVoltage),
	)
	ch <- prometheus.MustNewConstMetric(
		pe.panelCurrent,
		prometheus.GaugeValue,
		float64(status.ArrayCurrent),
	)
	ch <- prometheus.MustNewConstMetric(
		pe.panelPower,
		prometheus.GaugeValue,
		float64(status.ArrayPower),
	)

	ch <- prometheus.MustNewConstMetric(
		pe.chargingPower,
		prometheus.GaugeValue,
		float64(status.ChargingPower),
	)
	ch <- prometheus.MustNewConstMetric(
		pe.chargingCurrent,
		prometheus.GaugeValue,
		float64(status.ChargingCurrent),
	)

	ch <- prometheus.MustNewConstMetric(
		pe.batteryVoltage,
		prometheus.GaugeValue,
		float64(status.BatteryVoltage),
	)
	ch <- prometheus.MustNewConstMetric(
		pe.batterySOC,
		prometheus.GaugeValue,
		float64(status.BatterySOC),
	)
	ch <- prometheus.MustNewConstMetric(
		pe.batteryTemp,
		prometheus.GaugeValue,
		float64(status.BatteryTemp),
	)
	ch <- prometheus.MustNewConstMetric(
		pe.batteryMinVoltage,
		prometheus.GaugeValue,
		float64(status.BatteryMinVoltage),
	)
	ch <- prometheus.MustNewConstMetric(
		pe.batteryMaxVoltage,
		prometheus.GaugeValue,
		float64(status.BatteryMaxVoltage),
	)

	ch <- prometheus.MustNewConstMetric(
		pe.deviceTemp,
		prometheus.GaugeValue,
		float64(status.DeviceTemp),
	)

	ch <- prometheus.MustNewConstMetric(
		pe.energyGeneratedDaily,
		prometheus.GaugeValue,
		float64(status.EnergyGeneratedDaily),
	)

	ch <- prometheus.MustNewConstMetric(
		pe.chargingStatus,
		prometheus.GaugeValue,
		float64(status.ChargingStatus),
	)

	return nil
}