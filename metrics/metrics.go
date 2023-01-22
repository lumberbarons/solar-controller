package metrics

import (
	"encoding/binary"
	"github.com/gin-gonic/gin"
	"github.com/goburrow/modbus"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

const (
	namespace = "solar"
)

type SolarCollector struct {
	modbusClient modbus.Client

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

type ControllerStatus struct {
	ArrayVoltage           float32   `json:"arrayVoltage"`
	ArrayCurrent           float32   `json:"arrayCurrent"`
	ArrayPower             float32   `json:"arrayPower"`
	ChargingCurrent		   float32   `json:"chargingCurrent"`
	ChargingPower		   float32   `json:"chargingPower"`
	BatteryVoltage         float32   `json:"batteryVoltage"`
	BatterySOC             int32     `json:"batterySoc"`
	BatteryTemp            float32   `json:"batteryTemp"`
	BatteryMaxVoltage      float32   `json:"batteryMaxVoltage"`
	BatteryMinVoltage      float32   `json:"batteryMinVoltage"`
	DeviceTemp             float32   `json:"deviceTemp"`
	EnergyGeneratedDaily   float32   `json:"energyGeneratedDaily"`
	ChargingStatus		   int32     `json:"chargingStatus"`
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
}

func (sc *SolarCollector) MetricsGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		metrics, err := sc.getStatus()
		if err != nil {
			log.Error("failed to get metrics: ", err)
			c.Status(http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusOK, metrics)
	}
}

func (sc *SolarCollector) Describe(ch chan <- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		sc.panelVoltage,
	}

	for _, d := range ds {
		ch <- d
	}

	sc.scrapeFailures.Describe(ch)
}

func (sc *SolarCollector) Collect(ch chan<- prometheus.Metric) {
	if err := sc.collect(ch); err != nil {
		log.Printf("Error getting solar controller data: %s", err)
		sc.scrapeFailures.Inc()
		sc.scrapeFailures.Collect(ch)
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

func (sc *SolarCollector) collect(ch chan <- prometheus.Metric) error {
	defer end(start("metrics collection"))

	status, err := sc.getStatus()
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(
		sc.panelVoltage,
		prometheus.GaugeValue,
		float64(status.ArrayVoltage),
	)
	ch <- prometheus.MustNewConstMetric(
		sc.panelCurrent,
		prometheus.GaugeValue,
		float64(status.ArrayCurrent),
	)
	ch <- prometheus.MustNewConstMetric(
		sc.panelPower,
		prometheus.GaugeValue,
		float64(status.ArrayPower),
	)

	ch <- prometheus.MustNewConstMetric(
		sc.chargingPower,
		prometheus.GaugeValue,
		float64(status.ChargingPower),
	)
	ch <- prometheus.MustNewConstMetric(
		sc.chargingCurrent,
		prometheus.GaugeValue,
		float64(status.ChargingCurrent),
	)

	ch <- prometheus.MustNewConstMetric(
		sc.batteryVoltage,
		prometheus.GaugeValue,
		float64(status.BatteryVoltage),
	)
	ch <- prometheus.MustNewConstMetric(
		sc.batterySOC,
		prometheus.GaugeValue,
		float64(status.BatterySOC),
	)
	ch <- prometheus.MustNewConstMetric(
		sc.batteryTemp,
		prometheus.GaugeValue,
		float64(status.BatteryTemp),
	)
	ch <- prometheus.MustNewConstMetric(
		sc.batteryMinVoltage,
		prometheus.GaugeValue,
		float64(status.BatteryMinVoltage),
	)
	ch <- prometheus.MustNewConstMetric(
		sc.batteryMaxVoltage,
		prometheus.GaugeValue,
		float64(status.BatteryMaxVoltage),
	)

	ch <- prometheus.MustNewConstMetric(
		sc.deviceTemp,
		prometheus.GaugeValue,
		float64(status.DeviceTemp),
	)

	ch <- prometheus.MustNewConstMetric(
		sc.energyGeneratedDaily,
		prometheus.GaugeValue,
		float64(status.EnergyGeneratedDaily),
	)

	ch <- prometheus.MustNewConstMetric(
		sc.chargingStatus,
		prometheus.GaugeValue,
		float64(status.ChargingStatus),
	)

	return nil
}

func (sc *SolarCollector) getStatus() (c ControllerStatus, err error) {
	results, err := sc.getValueFloats(0x3100, 2)
	if err != nil {
		return c, err
	}

	c.ArrayVoltage = results[0]
	c.ArrayCurrent = results[0]

	c.ArrayCurrent, err = sc.getValueFloat(0x3101)
	if err != nil {
		return c, err
	}

	c.BatteryVoltage, err = sc.getValueFloat(0x3104)
	if err != nil {
		return c, err
	}

	c.BatterySOC, _ = sc.getValueInt(0x311A)
	if err != nil {
		return c, err
	}

	results, err = sc.getValueFloats(0x3302, 2)
	if err != nil {
		return c, err
	}

	c.BatteryMaxVoltage = results[0]
	c.BatteryMinVoltage = results[1]

	c.ArrayPower, err = sc.getValueFloat32(0x3102)
	if err != nil {
		return c, err
	}

	c.ChargingCurrent, err = sc.getValueFloat(0x3105)
	if err != nil {
		return c, err
	}

	c.ChargingPower, err = sc.getValueFloat32(0x3106)
	if err != nil {
		return c, err
	}

	c.EnergyGeneratedDaily, err = sc.getValueFloat32(0x330C)
	if err != nil {
		return c, err
	}

	controllerStatus, err := sc.getValueInt(0x3201)
	if err != nil {
		return c, err
	}

	chargingStatus := (controllerStatus & 0x0C) >> 2
	c.ChargingStatus = chargingStatus

	tempResults, err := sc.getValueInts(0x3110, 2)
	if err != nil {
		return c, err
	}

	bt := tempResults[0]

	if bt > 32768 {
		bt = bt - 65536
	}
	c.BatteryTemp = float32(bt) / 100

	dt := tempResults[1]

	if dt > 32768 {
		dt = dt - 65536
	}
	c.DeviceTemp = float32(dt) / 100

	return c, nil
}

func (sc *SolarCollector) getValueFloat(address uint16) (float32, error) {
	data, err := sc.modbusClient.ReadInputRegisters(address, 1)
	if err != nil {
		log.Warnf("Failed to get data, address: %d", address)
		return 0, err
	}

	return  float32(binary.BigEndian.Uint16(data)) / 100, nil
}

func (sc *SolarCollector) getValueFloats(address uint16, quantity uint16) ([]float32, error) {
	data, err := sc.modbusClient.ReadInputRegisters(address, quantity)
	if err != nil {
		log.Warnf("Failed to get data, address: %d", address)
		return nil, err
	}

	results := make([]float32, quantity)
	for i := 0; i < int(quantity); i++ {
		results[i] = float32(binary.BigEndian.Uint16(data[i * 2:i * 2 + 2])) / 100
	}

	return results, nil
}

func (sc *SolarCollector) getValueInt(address uint16) (int32, error) {
	data, err := sc.modbusClient.ReadInputRegisters(address, 1)
	if err != nil {
		log.Warnf("Failed to get data, address: %d", address)
		return 0, err
	}
	return int32(binary.BigEndian.Uint16(data)), nil
}

func (sc *SolarCollector) getValueInts(address uint16, quantity uint16) ([]int32, error) {
	data, err := sc.modbusClient.ReadInputRegisters(address, quantity)
	if err != nil {
		log.Warnf("Failed to get data, address: %d", address)
		return nil, err
	}

	results := make([]int32, quantity)
	for i := 0; i < int(quantity); i++ {
		results[i] = int32(binary.BigEndian.Uint16(data[i * 2:i * 2 + 2]))
	}

	return results, nil
}

func (sc *SolarCollector) getValueFloat32(address uint16) (float32, error) {
	data, err := sc.modbusClient.ReadInputRegisters(address, 2)

	if err != nil {
		log.Warnf("Failed to get data, address: %d", address)
		return 0, err
	}

	swappedData := append(data[2:4],data[0:2]...)
	return float32(binary.BigEndian.Uint32(swappedData)) / 100, nil
}
