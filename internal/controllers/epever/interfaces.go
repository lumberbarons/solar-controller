package epever

import (
	"context"
)

// ModbusClient defines the interface for Modbus communication operations.
// This abstraction allows for testing without physical hardware.
type ModbusClient interface {
	// ReadInputRegisters reads input registers from the Modbus device.
	ReadInputRegisters(ctx context.Context, address, quantity uint16) ([]byte, error)

	// ReadHoldingRegisters reads holding registers from the Modbus device.
	ReadHoldingRegisters(ctx context.Context, address, quantity uint16) ([]byte, error)

	// WriteSingleRegister writes a single holding register to the Modbus device.
	WriteSingleRegister(ctx context.Context, address, value uint16) ([]byte, error)

	// WriteMultipleRegisters writes multiple holding registers to the Modbus device.
	WriteMultipleRegisters(ctx context.Context, address, quantity uint16, value []byte) ([]byte, error)

	// ReadCoils reads coil status from the Modbus device.
	ReadCoils(ctx context.Context, address, quantity uint16) ([]byte, error)

	// WriteSingleCoil writes a single coil to the Modbus device.
	WriteSingleCoil(ctx context.Context, address, value uint16) ([]byte, error)

	// Close closes the Modbus client connection.
	Close()
}

// MetricsCollector defines the interface for collecting and exposing metrics.
// This abstraction allows for testing without the Prometheus global registry.
type MetricsCollector interface {
	// IncrementFailures increments the failure counter.
	IncrementFailures()

	// IncrementWriteFailures increments the write failure counter.
	IncrementWriteFailures()

	// IncrementRegisterFailure increments the register-specific failure counter.
	IncrementRegisterFailure(address uint16, registerType string)

	// SetMetrics updates all metrics based on the provided status.
	SetMetrics(status *ControllerStatus)
}
