package epever

import (
	"context"
	"fmt"
	"time"

	"github.com/lumberbarons/solar-controller/internal/controllers"
	"github.com/lumberbarons/solar-controller/internal/controllers/epever/parser"
	log "github.com/sirupsen/logrus"
)

// Modbus input register addresses (read-only status values)
const (
	regArrayVoltage         = 0x3100
	regArrayCurrent         = 0x3101
	regArrayPower           = 0x3102
	regBatteryVoltage       = 0x3104
	regChargingCurrent      = 0x3105
	regChargingPower        = 0x3106
	regBatteryTemperature   = 0x3110
	regDeviceTemperature    = 0x3111
	regBatterySOC           = 0x311A
	regControllerStatus     = 0x3201
	regBatteryMaxVoltage    = 0x3302
	regBatteryMinVoltage    = 0x3303
	regEnergyGeneratedDaily = 0x330C
)

// Charging status bit mask
const (
	chargingStatusMask  = 0x0C
	chargingStatusShift = 2
)

type Collector struct {
	modbusClient        controllers.ModbusClient
	prometheusCollector controllers.MetricsCollector
}

type ControllerStatus struct {
	Timestamp            int64   `json:"timestamp"`
	CollectionTime       float64 `json:"collectionTime"`
	ArrayVoltage         float32 `json:"arrayVoltage"`
	ArrayCurrent         float32 `json:"arrayCurrent"`
	ArrayPower           float32 `json:"arrayPower"`
	ChargingCurrent      float32 `json:"chargingCurrent"`
	ChargingPower        float32 `json:"chargingPower"`
	BatteryVoltage       float32 `json:"batteryVoltage"`
	BatterySOC           int32   `json:"batterySoc"`
	BatteryTemp          float32 `json:"batteryTemp"`
	BatteryMaxVoltage    float32 `json:"batteryMaxVoltage"`
	BatteryMinVoltage    float32 `json:"batteryMinVoltage"`
	DeviceTemp           float32 `json:"deviceTemp"`
	EnergyGeneratedDaily float32 `json:"energyGeneratedDaily"`
	ChargingStatus       int32   `json:"chargingStatus"`
}

func NewCollector(client controllers.ModbusClient, prometheusCollector controllers.MetricsCollector) *Collector {
	collector := &Collector{
		modbusClient:        client,
		prometheusCollector: prometheusCollector,
	}

	return collector
}

func (e *Collector) GetStatus(ctx context.Context) (*ControllerStatus, error) {
	startTime := time.Now()
	log.Debug("Starting GetStatus collection")

	c := &ControllerStatus{
		Timestamp: startTime.Unix(),
	}

	log.Debugf("Reading array voltage/current from register 0x%04X", regArrayVoltage)
	results, err := e.getValueFloats(ctx, regArrayVoltage, 2)
	if err != nil {
		log.Debugf("Failed to read array voltage/current: %v", err)
		return nil, err
	}
	if len(results) < 2 {
		return nil, fmt.Errorf("expected 2 values for array voltage/current, got %d", len(results))
	}
	log.Debugf("Array voltage: %.2fV, Array current: %.2fA", results[0], results[1])
	time.Sleep(100 * time.Millisecond) // Allow device to recover before next read

	c.ArrayVoltage = results[0]
	c.ArrayCurrent = results[1]

	log.Debugf("Reading battery voltage from register 0x%04X", regBatteryVoltage)
	c.BatteryVoltage, err = e.getValueFloat(ctx, regBatteryVoltage)
	if err != nil {
		log.Debugf("Failed to read battery voltage: %v", err)
		return nil, err
	}
	log.Debugf("Battery voltage: %.2fV", c.BatteryVoltage)
	time.Sleep(100 * time.Millisecond) // Allow device to recover before next read

	log.Debugf("Reading battery SOC from register 0x%04X", regBatterySOC)
	c.BatterySOC, err = e.getValueInt(ctx, regBatterySOC)
	if err != nil {
		log.Debugf("Failed to read battery SOC: %v", err)
		return nil, err
	}
	log.Debugf("Battery SOC: %d%%", c.BatterySOC)
	time.Sleep(100 * time.Millisecond) // Allow device to recover before next read

	log.Debugf("Reading battery max/min voltage from register 0x%04X", regBatteryMaxVoltage)
	results, err = e.getValueFloats(ctx, regBatteryMaxVoltage, 2)
	if err != nil {
		log.Debugf("Failed to read battery max/min voltage: %v", err)
		return nil, err
	}
	if len(results) < 2 {
		return nil, fmt.Errorf("expected 2 values for battery max/min voltage, got %d", len(results))
	}
	log.Debugf("Battery max voltage: %.2fV, Battery min voltage: %.2fV", results[0], results[1])
	time.Sleep(100 * time.Millisecond) // Allow device to recover before next read

	c.BatteryMaxVoltage = results[0]
	c.BatteryMinVoltage = results[1]

	log.Debugf("Reading array power from register 0x%04X", regArrayPower)
	c.ArrayPower, err = e.getValueFloat32(ctx, regArrayPower)
	if err != nil {
		log.Debugf("Failed to read array power: %v", err)
		return nil, err
	}
	log.Debugf("Array power: %.2fW", c.ArrayPower)
	time.Sleep(100 * time.Millisecond) // Allow device to recover before next read

	log.Debugf("Reading charging current from register 0x%04X", regChargingCurrent)
	c.ChargingCurrent, err = e.getValueFloat(ctx, regChargingCurrent)
	if err != nil {
		log.Debugf("Failed to read charging current: %v", err)
		return nil, err
	}
	log.Debugf("Charging current: %.2fA", c.ChargingCurrent)
	time.Sleep(100 * time.Millisecond) // Allow device to recover before next read

	log.Debugf("Reading charging power from register 0x%04X", regChargingPower)
	c.ChargingPower, err = e.getValueFloat32(ctx, regChargingPower)
	if err != nil {
		log.Debugf("Failed to read charging power: %v", err)
		return nil, err
	}
	log.Debugf("Charging power: %.2fW", c.ChargingPower)
	time.Sleep(100 * time.Millisecond) // Allow device to recover before next read

	log.Debugf("Reading energy generated daily from register 0x%04X", regEnergyGeneratedDaily)
	c.EnergyGeneratedDaily, err = e.getValueFloat32(ctx, regEnergyGeneratedDaily)
	if err != nil {
		log.Debugf("Failed to read energy generated daily: %v", err)
		return nil, err
	}
	log.Debugf("Energy generated daily: %.2fkWh", c.EnergyGeneratedDaily)
	time.Sleep(100 * time.Millisecond) // Allow device to recover before next read

	log.Debugf("Reading controller status from register 0x%04X", regControllerStatus)
	controllerStatus, err := e.getValueInt(ctx, regControllerStatus)
	if err != nil {
		log.Debugf("Failed to read controller status: %v", err)
		return nil, err
	}
	time.Sleep(100 * time.Millisecond) // Allow device to recover before next read

	chargingStatus := (controllerStatus & chargingStatusMask) >> chargingStatusShift
	c.ChargingStatus = chargingStatus
	log.Debugf("Controller status: 0x%04X, Charging status: %d", controllerStatus, chargingStatus)

	log.Debugf("Reading temperature data from register 0x%04X", regBatteryTemperature)
	tempData, err := e.modbusClient.ReadInputRegisters(ctx, regBatteryTemperature, 2)
	if err != nil {
		log.Debugf("Failed to read temperature data: %v", err)
		e.prometheusCollector.IncrementRegisterFailure(regBatteryTemperature, "input")
		return nil, fmt.Errorf("failed to read temperature data: %w", err)
	}
	// No delay needed after final read

	c.BatteryTemp, c.DeviceTemp, err = parser.ParseTemperatures(tempData)
	if err != nil {
		log.Debugf("Failed to parse temperatures: %v", err)
		return nil, fmt.Errorf("failed to parse temperatures: %w", err)
	}
	log.Debugf("Battery temp: %.1f°C, Device temp: %.1f°C", c.BatteryTemp, c.DeviceTemp)

	c.CollectionTime = time.Since(startTime).Seconds()
	log.Debugf("GetStatus completed in %.3fs", c.CollectionTime)

	return c, nil
}

func (e *Collector) getValueFloat(ctx context.Context, address uint16) (float32, error) {
	log.Debugf("Reading input register 0x%04X (quantity: 1)", address)
	data, err := e.modbusClient.ReadInputRegisters(ctx, address, 1)
	if err != nil {
		log.Debugf("ReadInputRegisters failed for 0x%04X: %v", address, err)
		e.prometheusCollector.IncrementRegisterFailure(address, "input")
		return 0, fmt.Errorf("failed to get data from address %d, error: %w", address, err)
	}
	log.Debugf("ReadInputRegisters 0x%04X returned %d bytes: %v", address, len(data), data)

	return parser.ParseFloat(data)
}

func (e *Collector) getValueFloats(ctx context.Context, address, quantity uint16) ([]float32, error) {
	log.Debugf("Reading input registers 0x%04X (quantity: %d)", address, quantity)
	data, err := e.modbusClient.ReadInputRegisters(ctx, address, quantity)
	if err != nil {
		log.Debugf("ReadInputRegisters failed for 0x%04X: %v", address, err)
		e.prometheusCollector.IncrementRegisterFailure(address, "input")
		return nil, fmt.Errorf("failed to get data from address %d, error: %w", address, err)
	}
	log.Debugf("ReadInputRegisters 0x%04X returned %d bytes: %v", address, len(data), data)

	return parser.ParseFloats(data, int(quantity))
}

func (e *Collector) getValueInt(ctx context.Context, address uint16) (int32, error) {
	log.Debugf("Reading input register 0x%04X (quantity: 1)", address)
	data, err := e.modbusClient.ReadInputRegisters(ctx, address, 1)
	if err != nil {
		log.Debugf("ReadInputRegisters failed for 0x%04X: %v", address, err)
		e.prometheusCollector.IncrementRegisterFailure(address, "input")
		return 0, fmt.Errorf("failed to get data from address %d, error: %w", address, err)
	}
	log.Debugf("ReadInputRegisters 0x%04X returned %d bytes: %v", address, len(data), data)
	return parser.ParseInt(data)
}

func (e *Collector) getValueFloat32(ctx context.Context, address uint16) (float32, error) {
	log.Debugf("Reading input registers 0x%04X (quantity: 2)", address)
	data, err := e.modbusClient.ReadInputRegisters(ctx, address, 2)
	if err != nil {
		log.Debugf("ReadInputRegisters failed for 0x%04X: %v", address, err)
		e.prometheusCollector.IncrementRegisterFailure(address, "input")
		return 0, fmt.Errorf("failed to get data from address %d, error: %w", address, err)
	}
	log.Debugf("ReadInputRegisters 0x%04X returned %d bytes: %v", address, len(data), data)

	return parser.ParseFloat32(data)
}
