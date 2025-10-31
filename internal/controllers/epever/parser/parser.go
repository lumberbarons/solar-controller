package parser

import (
	"encoding/binary"
	"fmt"
)

// Temperature conversion constants
const (
	TempSignedThreshold = 32768
	TempSignedOffset    = 65536
	TempDivisor         = 100.0
)

// Voltage/Current divisor (stored as centivolt/centiamp)
const ValueDivisor = 100.0

// ParseFloat parses a single register (2 bytes) as a float32 value.
// The value is divided by 100 to convert from centivolt/centiamp to volt/amp.
func ParseFloat(data []byte) (float32, error) {
	if len(data) < 2 {
		return 0, fmt.Errorf("insufficient data for float: expected 2 bytes, got %d", len(data))
	}
	return float32(binary.BigEndian.Uint16(data)) / ValueDivisor, nil
}

// ParseFloats parses multiple consecutive registers as float32 values.
// Each register (2 bytes) is divided by 100.
func ParseFloats(data []byte, quantity int) ([]float32, error) {
	expectedBytes := quantity * 2
	if len(data) < expectedBytes {
		return nil, fmt.Errorf("insufficient data for %d floats: expected %d bytes, got %d", quantity, expectedBytes, len(data))
	}

	results := make([]float32, quantity)
	for i := 0; i < quantity; i++ {
		results[i] = float32(binary.BigEndian.Uint16(data[i*2:i*2+2])) / ValueDivisor
	}

	return results, nil
}

// ParseInt parses a single register (2 bytes) as an int32 value.
func ParseInt(data []byte) (int32, error) {
	if len(data) < 2 {
		return 0, fmt.Errorf("insufficient data for int: expected 2 bytes, got %d", len(data))
	}
	return int32(binary.BigEndian.Uint16(data)), nil
}

// ParseInts parses multiple consecutive registers as int32 values.
func ParseInts(data []byte, quantity int) ([]int32, error) {
	expectedBytes := quantity * 2
	if len(data) < expectedBytes {
		return nil, fmt.Errorf("insufficient data for %d ints: expected %d bytes, got %d", quantity, expectedBytes, len(data))
	}

	results := make([]int32, quantity)
	for i := 0; i < quantity; i++ {
		results[i] = int32(binary.BigEndian.Uint16(data[i*2 : i*2+2]))
	}

	return results, nil
}

// ParseFloat32 parses two consecutive registers (4 bytes) as a float32 value with byte swapping.
// Epever devices store float32 values with swapped word order (low word first, high word second).
// The value is divided by 100 to convert from centivolt to volt.
func ParseFloat32(data []byte) (float32, error) {
	if len(data) < 4 {
		return 0, fmt.Errorf("insufficient data for float32: expected 4 bytes, got %d", len(data))
	}

	// Swap the word order: data[2:4] becomes first, data[0:2] becomes second
	swappedData := make([]byte, 4)
	copy(swappedData[0:2], data[2:4])
	copy(swappedData[2:4], data[0:2])

	return float32(binary.BigEndian.Uint32(swappedData)) / ValueDivisor, nil
}

// ParseSignedTemperature converts a raw temperature value to a signed float32.
// Epever devices use an unsigned representation where values >= 32768 represent negative temperatures.
// The final value is divided by 100 to convert from centi-degrees to degrees.
func ParseSignedTemperature(raw int32) float32 {
	temp := raw
	if temp >= TempSignedThreshold {
		temp -= TempSignedOffset
	}
	return float32(temp) / TempDivisor
}

// ParseTemperatures parses two consecutive temperature registers.
// Returns battery temperature and device temperature.
func ParseTemperatures(data []byte) (batteryTemp, deviceTemp float32, err error) {
	temps, err := ParseInts(data, 2)
	if err != nil {
		return 0, 0, err
	}
	if len(temps) < 2 {
		return 0, 0, fmt.Errorf("expected 2 temperature values, got %d", len(temps))
	}

	batteryTemp = ParseSignedTemperature(temps[0])
	deviceTemp = ParseSignedTemperature(temps[1])

	return batteryTemp, deviceTemp, nil
}

// EncodeUint16 encodes a uint16 value to bytes in big-endian order.
func EncodeUint16(value uint16) []byte {
	data := make([]byte, 2)
	binary.BigEndian.PutUint16(data, value)
	return data
}

// EncodeUint16s encodes multiple uint16 values to bytes in big-endian order.
func EncodeUint16s(values []uint16) []byte {
	data := make([]byte, len(values)*2)
	for i, v := range values {
		binary.BigEndian.PutUint16(data[i*2:], v)
	}
	return data
}

// EncodeVoltage encodes a voltage value (in volts) to a uint16 (in centivolts).
func EncodeVoltage(voltage float32) uint16 {
	return uint16(voltage * ValueDivisor)
}

// EncodeTemperature encodes a temperature value (in degrees) to an int16 (in centi-degrees).
func EncodeTemperature(temp float32) int16 {
	return int16(temp * TempDivisor)
}
