package controllers

import (
	"context"

	"github.com/gin-gonic/gin"
)

// SolarController defines the interface that all solar equipment controllers must implement.
// Controllers manage the lifecycle of hardware communication, metrics collection, and API endpoints.
type SolarController interface {
	// RegisterEndpoints registers HTTP endpoints for this controller.
	RegisterEndpoints(r *gin.Engine)

	// Enabled returns whether this controller is enabled and should be started.
	Enabled() bool

	// Close performs cleanup and releases resources held by the controller.
	Close() error
}

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

// MessagePublisher defines the interface for publishing messages to a message broker.
// This abstraction allows for testing without a real MQTT broker.
type MessagePublisher interface {
	// Publish publishes a message with the given topic suffix and payload.
	Publish(topicSuffix, payload string)

	// Close closes the publisher connection.
	Close()
}

// MetricsCollector defines the interface for collecting and exposing metrics.
// This abstraction allows for testing without the Prometheus global registry.
type MetricsCollector interface {
	// IncrementFailures increments the failure counter.
	IncrementFailures()

	// IncrementWriteFailures increments the write failure counter (if supported).
	IncrementWriteFailures()

	// SetMetrics updates all metrics based on the provided status.
	// The status parameter should be a pointer to a controller-specific status struct.
	SetMetrics(status interface{})
}

