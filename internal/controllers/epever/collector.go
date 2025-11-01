package epever

import (
	"context"
	"fmt"
	"time"

	"github.com/lumberbarons/solar-controller/internal/controllers"
	"github.com/lumberbarons/solar-controller/internal/controllers/epever/parser"
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
	modbusClient controllers.ModbusClient
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

func NewCollector(client controllers.ModbusClient) *Collector {
	collector := &Collector{
		modbusClient: client,
	}

	return collector
}

func (e *Collector) GetStatus(ctx context.Context) (*ControllerStatus, error) {
	startTime := time.Now()

	c := &ControllerStatus{
		Timestamp: startTime.Unix(),
	}

	results, err := e.getValueFloats(ctx, regArrayVoltage, 2)
	if err != nil {
		return nil, err
	}
	if len(results) < 2 {
		return nil, fmt.Errorf("expected 2 values for array voltage/current, got %d", len(results))
	}
	time.Sleep(50 * time.Millisecond) // Allow device to recover before next read

	c.ArrayVoltage = results[0]
	c.ArrayCurrent = results[1]

	c.BatteryVoltage, err = e.getValueFloat(ctx, regBatteryVoltage)
	if err != nil {
		return nil, err
	}
	time.Sleep(50 * time.Millisecond) // Allow device to recover before next read

	c.BatterySOC, err = e.getValueInt(ctx, regBatterySOC)
	if err != nil {
		return nil, err
	}
	time.Sleep(50 * time.Millisecond) // Allow device to recover before next read

	results, err = e.getValueFloats(ctx, regBatteryMaxVoltage, 2)
	if err != nil {
		return nil, err
	}
	if len(results) < 2 {
		return nil, fmt.Errorf("expected 2 values for battery max/min voltage, got %d", len(results))
	}
	time.Sleep(50 * time.Millisecond) // Allow device to recover before next read

	c.BatteryMaxVoltage = results[0]
	c.BatteryMinVoltage = results[1]

	c.ArrayPower, err = e.getValueFloat32(ctx, regArrayPower)
	if err != nil {
		return nil, err
	}
	time.Sleep(50 * time.Millisecond) // Allow device to recover before next read

	c.ChargingCurrent, err = e.getValueFloat(ctx, regChargingCurrent)
	if err != nil {
		return nil, err
	}
	time.Sleep(50 * time.Millisecond) // Allow device to recover before next read

	c.ChargingPower, err = e.getValueFloat32(ctx, regChargingPower)
	if err != nil {
		return nil, err
	}
	time.Sleep(50 * time.Millisecond) // Allow device to recover before next read

	c.EnergyGeneratedDaily, err = e.getValueFloat32(ctx, regEnergyGeneratedDaily)
	if err != nil {
		return nil, err
	}
	time.Sleep(50 * time.Millisecond) // Allow device to recover before next read

	controllerStatus, err := e.getValueInt(ctx, regControllerStatus)
	if err != nil {
		return nil, err
	}
	time.Sleep(50 * time.Millisecond) // Allow device to recover before next read

	chargingStatus := (controllerStatus & chargingStatusMask) >> chargingStatusShift
	c.ChargingStatus = chargingStatus

	tempData, err := e.modbusClient.ReadInputRegisters(ctx, regBatteryTemperature, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to read temperature data: %w", err)
	}
	// No delay needed after final read

	c.BatteryTemp, c.DeviceTemp, err = parser.ParseTemperatures(tempData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse temperatures: %w", err)
	}

	c.CollectionTime = time.Since(startTime).Seconds()

	return c, nil
}

func (e *Collector) getValueFloat(ctx context.Context, address uint16) (float32, error) {
	data, err := e.modbusClient.ReadInputRegisters(ctx, address, 1)
	if err != nil {
		return 0, fmt.Errorf("failed to get data from address %d, error: %w", address, err)
	}

	return parser.ParseFloat(data)
}

func (e *Collector) getValueFloats(ctx context.Context, address, quantity uint16) ([]float32, error) {
	data, err := e.modbusClient.ReadInputRegisters(ctx, address, quantity)
	if err != nil {
		return nil, fmt.Errorf("failed to get data from address %d, error: %w", address, err)
	}

	return parser.ParseFloats(data, int(quantity))
}

func (e *Collector) getValueInt(ctx context.Context, address uint16) (int32, error) {
	data, err := e.modbusClient.ReadInputRegisters(ctx, address, 1)
	if err != nil {
		return 0, fmt.Errorf("failed to get data from address %d, error: %w", address, err)
	}
	return parser.ParseInt(data)
}

func (e *Collector) getValueFloat32(ctx context.Context, address uint16) (float32, error) {
	data, err := e.modbusClient.ReadInputRegisters(ctx, address, 2)
	if err != nil {
		return 0, fmt.Errorf("failed to get data from address %d, error: %w", address, err)
	}

	return parser.ParseFloat32(data)
}
