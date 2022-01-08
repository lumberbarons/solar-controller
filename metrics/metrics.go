package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

const (
	namespace = "solar"
)

type SolarCollector struct {
	mutex sync.Mutex

	portName string

	scrapeFailures prometheus.Counter

	panelVoltage *prometheus.Desc
	panelCurrent *prometheus.Desc
	panelPower   *prometheus.Desc

	batteryVoltage    *prometheus.Desc
	batterySOC        *prometheus.Desc
	batteryTemp       *prometheus.Desc
	batteryMaxVoltage *prometheus.Desc
	batteryMinVoltage *prometheus.Desc

	deviceTemp *prometheus.Desc

	energyGeneratedDaily   *prometheus.Desc
	energyGeneratedMonthly *prometheus.Desc
	energyGeneratedAnnual  *prometheus.Desc
	energyGeneratedTotal   *prometheus.Desc
}

func NewMetricsCollector(portName string) *SolarCollector {
	return &SolarCollector{
		portName: portName,

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
			"Controller calculated daily power generation, (kWh)",
			nil,
			nil,
		),
		energyGeneratedMonthly: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "energy_generated_monthly"),
			"Controller calculated monthly power generation, (kWh)",
			nil,
			nil,
		),
		energyGeneratedAnnual: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "energy_generated_annual"),
			"Controller calculated annual power generation, (kWh)",
			nil,
			nil,
		),
		energyGeneratedTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "energy_generated_total"),
			"Controller calculated total power generation, (kWh)",
			nil,
			nil,
		),
	}
}

func (c *SolarCollector) Describe(ch chan <- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		c.panelVoltage,
	}

	for _, d := range ds {
		ch <- d
	}

	c.scrapeFailures.Describe(ch)
}

func (c *SolarCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock() // To protect metrics from concurrent collects.
	defer c.mutex.Unlock()
	if err := c.collect(ch); err != nil {
		log.Printf("Error getting solar controller data: %s", err)
		c.scrapeFailures.Inc()
		c.scrapeFailures.Collect(ch)
	}
	return
}

func start(s string) (string, time.Time) {
	log.Printf("Start %s", s)
	return s, time.Now()
}

func end(s string, startTime time.Time) {
	endTime := time.Now()
	log.Printf("End %s, time: %.4f sec", s, endTime.Sub(startTime).Seconds())
}

func (c *SolarCollector) collect(ch chan <- prometheus.Metric) error {
	defer end(start("metrics collection"))

	status, err := getStatus(c.portName)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(
		c.panelVoltage,
		prometheus.GaugeValue,
		float64(status.ArrayVoltage),
	)
	ch <- prometheus.MustNewConstMetric(
		c.panelCurrent,
		prometheus.GaugeValue,
		float64(status.ArrayCurrent),
	)
	ch <- prometheus.MustNewConstMetric(
		c.panelPower,
		prometheus.GaugeValue,
		float64(status.ArrayPower),
	)

	ch <- prometheus.MustNewConstMetric(
		c.batteryVoltage,
		prometheus.GaugeValue,
		float64(status.BatteryVoltage),
	)
	ch <- prometheus.MustNewConstMetric(
		c.batterySOC,
		prometheus.GaugeValue,
		float64(status.BatterySOC),
	)
	ch <- prometheus.MustNewConstMetric(
		c.batteryTemp,
		prometheus.GaugeValue,
		float64(status.BatteryTemp),
	)
	ch <- prometheus.MustNewConstMetric(
		c.batteryMinVoltage,
		prometheus.GaugeValue,
		float64(status.BatteryMinVoltage),
	)
	ch <- prometheus.MustNewConstMetric(
		c.batteryMaxVoltage,
		prometheus.GaugeValue,
		float64(status.BatteryMaxVoltage),
	)

	ch <- prometheus.MustNewConstMetric(
		c.deviceTemp,
		prometheus.GaugeValue,
		float64(status.DeviceTemp),
	)

	ch <- prometheus.MustNewConstMetric(
		c.energyGeneratedDaily,
		prometheus.GaugeValue,
		float64(status.EnergyGeneratedDaily),
	)
	ch <- prometheus.MustNewConstMetric(
		c.energyGeneratedMonthly,
		prometheus.GaugeValue,
		float64(status.EnergyGeneratedMonthly),
	)
	ch <- prometheus.MustNewConstMetric(
		c.energyGeneratedAnnual,
		prometheus.GaugeValue,
		float64(status.EnergyGeneratedAnnual),
	)
	ch <- prometheus.MustNewConstMetric(
		c.energyGeneratedTotal,
		prometheus.GaugeValue,
		float64(status.EnergyGeneratedTotal),
	)

	return nil
}
