package metrics

import (
	"encoding/binary"
	"github.com/goburrow/modbus"
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

	modbusClient modbus.Client

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

type ControllerStatus struct {
	ArrayVoltage           float32   `json:"arrayVoltage"`
	ArrayCurrent           float32   `json:"arrayCurrent"`
	ArrayPower             float32   `json:"arrayPower"`
	BatteryVoltage         float32   `json:"batteryVoltage"`
	BatterySOC             int32     `json:"batterySoc"`
	BatteryTemp            float32   `json:"batteryTemp"`
	BatteryMaxVoltage      float32   `json:"batteryMaxVoltage"`
	BatteryMinVoltage      float32   `json:"batteryMinVoltage"`
	DeviceTemp             float32   `json:"deviceTemp"`
	EnergyGeneratedDaily   float32   `json:"energyGeneratedDaily"`
	EnergyGeneratedMonthly float32   `json:"energyGeneratedMonthly"`
	EnergyGeneratedAnnual  float32   `json:"energyGeneratedAnnually"`
	EnergyGeneratedTotal   float32   `json:"energyGeneratedTotal"`
}

func NewSolarCollector(client modbus.Client) *SolarCollector {
	return &SolarCollector{
		modbusClient: client,

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
	log.Printf("start %s", s)
	return s, time.Now()
}


func end(s string, startTime time.Time) {
	endTime := time.Now()
	log.Printf("end %s, time: %.4f sec", s, endTime.Sub(startTime).Seconds())
}

func (c *SolarCollector) collect(ch chan <- prometheus.Metric) error {
	defer end(start("metrics collection"))

	status, err := getStatus(c.modbusClient)
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

func getStatus(client modbus.Client) (c ControllerStatus, err error) {
	c.ArrayVoltage = getValue(client, 0x3100) / 100
	c.ArrayCurrent = getValue(client, 0x3101) / 100
	c.BatteryVoltage = getValue(client, 0x3104) / 100
	c.BatterySOC = int32(getValue(client, 0x311A))

	c.BatteryMaxVoltage = getValue(client, 0x3302) / 100
	c.BatteryMinVoltage = getValue(client, 0x3303) / 100

	c.ArrayPower = getValue32(client, 0x3102) / 100

	c.EnergyGeneratedDaily = getValue32(client, 0x330C) / 100
	c.EnergyGeneratedMonthly = getValue32(client, 0x330E) / 100
	c.EnergyGeneratedAnnual = getValue32(client, 0x3310) / 100
	c.EnergyGeneratedTotal = getValue32(client, 0x3312) / 100

	bt := getValue(client, 0x3110)
	if bt > 32768 {
		bt = bt - 65536
	}
	c.BatteryTemp = bt / 100

	dt := getValue(client, 0x3111)
	if dt > 32768 {
		dt = dt - 65536
	}
	c.DeviceTemp = dt / 100

	return
}

func getValue(client modbus.Client, address uint16) float32 {
	data, err := client.ReadInputRegisters(address, 1)
	if err != nil {
		log.Warnf("failed to get data, address: %d", address)
		return 0 // todo
	}
	return float32(binary.BigEndian.Uint16(data))
}

func getValue32(client modbus.Client, lowAddress uint16) float32 {
	lowData, err := client.ReadInputRegisters(lowAddress, 1)
	if err != nil {
		log.Warnf("failed to get data, address: %d", lowAddress)
		return 0 // todo
	}

	highAddress := lowAddress + 1

	highData, err := client.ReadInputRegisters(highAddress, 1)
	if err != nil {
		log.Warnf("failed to get data, address: %d", highAddress)
		return 0 // todo
	}

	swappedData := append(highData,lowData...)
	return float32(binary.BigEndian.Uint32(swappedData))
}
