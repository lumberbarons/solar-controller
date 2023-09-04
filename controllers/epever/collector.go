package epever

import (
	"encoding/binary"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/goburrow/modbus"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
	"time"
)

const (
	topicSuffix = "epever"
)

type Collector struct {
	mu sync.Mutex
	modbusClient modbus.Client

	cacheExpiry    int64
	cacheTimestamp int64
	cachedMetrics  *ControllerStatus
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

func NewCollector(client modbus.Client, cacheExpiry int64) *Collector {
	collector := &Collector{
		modbusClient: client,
		cacheExpiry: cacheExpiry,
	}

	return collector
}

func (e *Collector) MetricsGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		metrics, err := e.GetStatus()
		if err != nil {
			log.Error("failed to get metrics: ", err)
			c.Status(http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusOK, metrics)
	}
}

func (e *Collector) GetStatus() (*ControllerStatus, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.cacheTimestamp < time.Now().Unix() - e.cacheExpiry {
		log.Info("epever cache expired, collecting metrics")

		metrics, err := e.collectMetrics()
		if err != nil {
			return nil, err
		}

		e.cacheTimestamp = metrics.Timestamp
		e.cachedMetrics = metrics
	}

	return e.cachedMetrics, nil
}

func (e *Collector) GetStatusString() (string, error) {
	status, err := e.GetStatus()
	if err != nil {
		return "", err
	}

	b, err := json.Marshal(status)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func (e *Collector) GetTopicSuffix() string {
	return topicSuffix
}

func (e *Collector) collectMetrics() (*ControllerStatus, error) {
	startTime := time.Now()

	c := &ControllerStatus{
		Timestamp: startTime.Unix(),
	}

	results, err := e.getValueFloats(0x3100, 2)
	if err != nil {
		return nil, err
	}

	c.ArrayVoltage = results[0]
	c.ArrayCurrent = results[0]

	c.ArrayCurrent, err = e.getValueFloat(0x3101)
	if err != nil {
		return nil, err
	}

	c.BatteryVoltage, err = e.getValueFloat(0x3104)
	if err != nil {
		return nil, err
	}

	c.BatterySOC, _ = e.getValueInt(0x311A)
	if err != nil {
		return nil, err
	}

	results, err = e.getValueFloats(0x3302, 2)
	if err != nil {
		return nil, err
	}

	c.BatteryMaxVoltage = results[0]
	c.BatteryMinVoltage = results[1]

	c.ArrayPower, err = e.getValueFloat32(0x3102)
	if err != nil {
		return nil, err
	}

	c.ChargingCurrent, err = e.getValueFloat(0x3105)
	if err != nil {
		return nil, err
	}

	c.ChargingPower, err = e.getValueFloat32(0x3106)
	if err != nil {
		return nil, err
	}

	c.EnergyGeneratedDaily, err = e.getValueFloat32(0x330C)
	if err != nil {
		return nil, err
	}

	controllerStatus, err := e.getValueInt(0x3201)
	if err != nil {
		return nil, err
	}

	chargingStatus := (controllerStatus & 0x0C) >> 2
	c.ChargingStatus = chargingStatus

	tempResults, err := e.getValueInts(0x3110, 2)
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

func (e *Collector) getValueFloat(address uint16) (float32, error) {
	data, err := e.modbusClient.ReadInputRegisters(address, 1)
	if err != nil {
		log.Warnf("Failed to get data, address: %d", address)
		return 0, err
	}

	return  float32(binary.BigEndian.Uint16(data)) / 100, nil
}

func (e *Collector) getValueFloats(address uint16, quantity uint16) ([]float32, error) {
	data, err := e.modbusClient.ReadInputRegisters(address, quantity)
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

func (e *Collector) getValueInt(address uint16) (int32, error) {
	data, err := e.modbusClient.ReadInputRegisters(address, 1)
	if err != nil {
		log.Warnf("Failed to get data, address: %d", address)
		return 0, err
	}
	return int32(binary.BigEndian.Uint16(data)), nil
}

func (e *Collector) getValueInts(address uint16, quantity uint16) ([]int32, error) {
	data, err := e.modbusClient.ReadInputRegisters(address, quantity)
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

func (e *Collector) getValueFloat32(address uint16) (float32, error) {
	data, err := e.modbusClient.ReadInputRegisters(address, 2)

	if err != nil {
		log.Warnf("Failed to get data, address: %d", address)
		return 0, err
	}

	swappedData := append(data[2:4],data[0:2]...)
	return float32(binary.BigEndian.Uint32(swappedData)) / 100, nil
}
