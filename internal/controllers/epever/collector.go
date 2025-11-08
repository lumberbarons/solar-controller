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
	regArrayVoltage       = 0x3100
	regArrayCurrent       = 0x3101
	regArrayPower         = 0x3102
	regBatteryVoltage     = 0x3104
	regChargingCurrent    = 0x3105
	regChargingPower      = 0x3106
	regBatteryTemperature = 0x3110
	regDeviceTemperature  = 0x3111

	regBatterySOC           = 0x311A
	regControllerStatus     = 0x3201
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

	// Batch read all PV, battery, charging, and temperature registers (0x3100-0x3111)
	log.Debugf("Reading batch data from registers 0x%04X-0x%04X (18 registers)", regArrayVoltage, regDeviceTemperature)
	batchData, err := e.modbusClient.ReadInputRegisters(ctx, regArrayVoltage, 18)
	if err != nil {
		log.Debugf("Failed to read batch data: %v", err)
		e.prometheusCollector.IncrementRegisterFailure(regArrayVoltage, "input")
		return nil, fmt.Errorf("failed to read batch data (0x%04X-0x%04X): %w", regArrayVoltage, regDeviceTemperature, err)
	}
	if len(batchData) < 36 {
		return nil, fmt.Errorf("insufficient batch data: expected 36 bytes, got %d", len(batchData))
	}
	time.Sleep(100 * time.Millisecond) // Allow device to recover after large read

	// Extract values from batch data
	// Register offsets within the batch (each register is 2 bytes):
	// 0x3100 (offset 0): Array voltage
	// 0x3101 (offset 2): Array current
	// 0x3102-0x3103 (offset 4-8): Array power (32-bit)
	// 0x3104 (offset 8): Battery voltage
	// 0x3105 (offset 10): Charging current
	// 0x3106-0x3107 (offset 12-16): Charging power (32-bit)
	// 0x3108-0x310F (offset 16-32): Unused registers (skipped)
	// 0x3110 (offset 32): Battery temperature
	// 0x3111 (offset 34): Device temperature

	c.ArrayVoltage, err = parser.ParseFloat(batchData[0:2])
	if err != nil {
		return nil, fmt.Errorf("failed to parse array voltage: %w", err)
	}

	c.ArrayCurrent, err = parser.ParseFloat(batchData[2:4])
	if err != nil {
		return nil, fmt.Errorf("failed to parse array current: %w", err)
	}

	c.ArrayPower, err = parser.ParseFloat32(batchData[4:8])
	if err != nil {
		return nil, fmt.Errorf("failed to parse array power: %w", err)
	}

	c.BatteryVoltage, err = parser.ParseFloat(batchData[8:10])
	if err != nil {
		return nil, fmt.Errorf("failed to parse battery voltage: %w", err)
	}

	c.ChargingCurrent, err = parser.ParseFloat(batchData[10:12])
	if err != nil {
		return nil, fmt.Errorf("failed to parse charging current: %w", err)
	}

	c.ChargingPower, err = parser.ParseFloat32(batchData[12:16])
	if err != nil {
		return nil, fmt.Errorf("failed to parse charging power: %w", err)
	}

	c.BatteryTemp, c.DeviceTemp, err = parser.ParseTemperatures(batchData[32:36])
	if err != nil {
		return nil, fmt.Errorf("failed to parse temperatures: %w", err)
	}

	log.Debugf("Batch data parsed - Array: %.2fV/%.2fA/%.2fW, Battery: %.2fV/%.1f°C, Charging: %.2fA/%.2fW, Device: %.1f°C",
		c.ArrayVoltage, c.ArrayCurrent, c.ArrayPower, c.BatteryVoltage, c.BatteryTemp,
		c.ChargingCurrent, c.ChargingPower, c.DeviceTemp)

	// Read remaining registers individually
	log.Debugf("Reading battery SOC from register 0x%04X", regBatterySOC)
	c.BatterySOC, err = e.getValueInt(ctx, regBatterySOC)
	if err != nil {
		log.Debugf("Failed to read battery SOC: %v", err)
		return nil, err
	}
	log.Debugf("Battery SOC: %d%%", c.BatterySOC)
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
	// No delay needed after final read

	chargingStatus := (controllerStatus & chargingStatusMask) >> chargingStatusShift
	c.ChargingStatus = chargingStatus
	log.Debugf("Controller status: 0x%04X, Charging status: %d", controllerStatus, chargingStatus)

	c.CollectionTime = time.Since(startTime).Seconds()
	log.Debugf("GetStatus completed in %.3fs", c.CollectionTime)

	return c, nil
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
