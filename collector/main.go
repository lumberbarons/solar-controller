package collector

import (
	"encoding/binary"
	"github.com/gin-gonic/gin"
	"github.com/goburrow/modbus"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
	"time"
)

type SolarCollector struct {
	mu sync.Mutex
	modbusClient modbus.Client

	collectionPeriod    int64
	collectionTimestamp int64
	cachedMetrics       *ControllerStatus
}

type ControllerStatus struct {
	Timestamp              int64     `json:"timestamp"`
	CollectionTime         float64   `json:"collectionTime"`
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

func NewSolarCollector(client modbus.Client, collectionPeriod int64) *SolarCollector {
	collector := &SolarCollector{
		modbusClient: client,
		collectionPeriod: collectionPeriod,
	}

	return collector
}

func (sc *SolarCollector) MetricsGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		metrics, err := sc.GetStatus()
		if err != nil {
			log.Error("failed to get metrics: ", err)
			c.Status(http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusOK, metrics)
	}
}

func (sc *SolarCollector) GetStatus() (*ControllerStatus, error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.collectionTimestamp < time.Now().Unix() - sc.collectionPeriod {
		log.Info("cache expired, collecting metrics")

		metrics, err := sc.collectMetrics()
		if err != nil {
			return nil, err
		}

		sc.collectionTimestamp = metrics.Timestamp
		sc.cachedMetrics = metrics
	}

	return sc.cachedMetrics, nil
}

func (sc *SolarCollector) collectMetrics() (*ControllerStatus, error) {
	startTime := time.Now()

	c := &ControllerStatus{
		Timestamp: startTime.Unix(),
	}

	results, err := sc.getValueFloats(0x3100, 2)
	if err != nil {
		return nil, err
	}

	c.ArrayVoltage = results[0]
	c.ArrayCurrent = results[0]

	c.ArrayCurrent, err = sc.getValueFloat(0x3101)
	if err != nil {
		return nil, err
	}

	c.BatteryVoltage, err = sc.getValueFloat(0x3104)
	if err != nil {
		return nil, err
	}

	c.BatterySOC, _ = sc.getValueInt(0x311A)
	if err != nil {
		return nil, err
	}

	results, err = sc.getValueFloats(0x3302, 2)
	if err != nil {
		return nil, err
	}

	c.BatteryMaxVoltage = results[0]
	c.BatteryMinVoltage = results[1]

	c.ArrayPower, err = sc.getValueFloat32(0x3102)
	if err != nil {
		return nil, err
	}

	c.ChargingCurrent, err = sc.getValueFloat(0x3105)
	if err != nil {
		return nil, err
	}

	c.ChargingPower, err = sc.getValueFloat32(0x3106)
	if err != nil {
		return nil, err
	}

	c.EnergyGeneratedDaily, err = sc.getValueFloat32(0x330C)
	if err != nil {
		return nil, err
	}

	controllerStatus, err := sc.getValueInt(0x3201)
	if err != nil {
		return nil, err
	}

	chargingStatus := (controllerStatus & 0x0C) >> 2
	c.ChargingStatus = chargingStatus

	tempResults, err := sc.getValueInts(0x3110, 2)
	if err != nil {
		return nil, err
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

	c .CollectionTime = time.Now().Sub(startTime).Seconds()

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
