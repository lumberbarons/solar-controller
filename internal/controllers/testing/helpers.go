package testing

import (
	"context"
	"encoding/binary"
	"math"
)

// CreateModbusResponse creates a Modbus response byte slice from uint16 values.
// This is useful for creating test data that mimics actual Modbus responses.
func CreateModbusResponse(values ...uint16) []byte {
	data := make([]byte, len(values)*2)
	for i, v := range values {
		binary.BigEndian.PutUint16(data[i*2:], v)
	}
	return data
}

// CreateModbusFloat32Response creates a Modbus response for a float32 value.
// Epever devices use a specific byte order for float32 values.
func CreateModbusFloat32Response(value float32) []byte {
	data := make([]byte, 4)
	bits := math.Float32bits(value)

	// Epever uses swapped byte order for float32
	binary.BigEndian.PutUint16(data[0:2], uint16(bits>>16))
	binary.BigEndian.PutUint16(data[2:4], uint16(bits&0xFFFF))

	return data
}

// ModbusResponseBuilder helps build complex Modbus responses for testing.
type ModbusResponseBuilder struct {
	data []byte
}

// NewModbusResponseBuilder creates a new ModbusResponseBuilder.
func NewModbusResponseBuilder() *ModbusResponseBuilder {
	return &ModbusResponseBuilder{
		data: make([]byte, 0),
	}
}

// AddUint16 adds a uint16 value to the response.
func (b *ModbusResponseBuilder) AddUint16(value uint16) *ModbusResponseBuilder {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, value)
	b.data = append(b.data, buf...)
	return b
}

// AddFloat32 adds a float32 value to the response (Epever byte order).
func (b *ModbusResponseBuilder) AddFloat32(value float32) *ModbusResponseBuilder {
	bits := math.Float32bits(value)
	b.AddUint16(uint16(bits >> 16))
	b.AddUint16(uint16(bits & 0xFFFF))
	return b
}

// AddInt16 adds a signed int16 value to the response.
func (b *ModbusResponseBuilder) AddInt16(value int16) *ModbusResponseBuilder {
	return b.AddUint16(uint16(value))
}

// Build returns the final byte slice.
func (b *ModbusResponseBuilder) Build() []byte {
	return b.data
}

// CreateModbusError creates a function that returns an error for testing error paths.
func CreateModbusError(message string) func(context.Context, uint16, uint16) ([]byte, error) {
	return func(context.Context, uint16, uint16) ([]byte, error) {
		return nil, &ModbusTestError{Message: message}
	}
}

// ModbusTestError is a simple error type for testing.
type ModbusTestError struct {
	Message string
}

func (e *ModbusTestError) Error() string {
	return e.Message
}
