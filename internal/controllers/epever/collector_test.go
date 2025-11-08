package epever

import (
	"context"
	"testing"

	testingpkg "github.com/lumberbarons/solar-controller/internal/controllers/testing"
)

func TestCollector_GetStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("successful status collection", func(t *testing.T) {
		mockClient := &testingpkg.MockModbusClient{
			ReadInputRegistersFunc: func(_ context.Context, address, quantity uint16) ([]byte, error) {
				// Return realistic test data based on the address and quantity
				switch address {
				case regArrayVoltage: // Batch read (18 registers: 0x3100-0x3111)
					if quantity == 18 {
						// Create 18 registers of data (36 bytes)
						// 0x3100: Array voltage (1850 = 18.5V)
						// 0x3101: Array current (520 = 5.2A)
						// 0x3102-0x3103: Array power (0, 962 = ~96.2W as 32-bit)
						// 0x3104: Battery voltage (1280 = 12.8V)
						// 0x3105: Charging current (480 = 4.8A)
						// 0x3106-0x3107: Charging power (0, 614 = ~61.4W as 32-bit)
						// 0x3108-0x310F: Unused (8 registers, set to 0)
						// 0x3110: Battery temp (2500 = 25°C)
						// 0x3111: Device temp (3200 = 32°C)
						return testingpkg.CreateModbusResponse(
							1850, 520, // Array V/I
							0, 962, // Array power (32-bit)
							1280,   // Battery voltage
							480,    // Charging current
							0, 614, // Charging power (32-bit)
							0, 0, 0, 0, 0, 0, 0, 0, // Unused registers (0x3108-0x310F)
							2500, 3200, // Battery temp, Device temp
						), nil
					}
					return testingpkg.CreateModbusResponse(1850, 520), nil // Legacy fallback
				case regBatterySOC: // Battery SOC (1 register)
					return testingpkg.CreateModbusResponse(85), nil // 85%
				case regEnergyGeneratedDaily: // Daily energy (2 registers for 32-bit)
					return testingpkg.CreateModbusResponse(0, 1550), nil // 15.5 kWh
				case regControllerStatus: // Controller status (1 register)
					return testingpkg.CreateModbusResponse(0x0004), nil // Charging status bits
				default:
					t.Fatalf("unexpected register address: 0x%X", address)
					return nil, nil
				}
			},
		}

		mockMetrics := &testingpkg.MockMetricsCollector{}
		collector := NewCollector(mockClient, mockMetrics)
		status, err := collector.GetStatus(ctx)

		if err != nil {
			t.Fatalf("GetStatus() error = %v", err)
		}

		if status == nil {
			t.Fatal("GetStatus() returned nil status")
		}

		// Verify all collected values
		tests := []struct {
			name string
			got  float32
			want float32
		}{
			{"ArrayVoltage", status.ArrayVoltage, 18.5},
			{"ArrayCurrent", status.ArrayCurrent, 5.2},
			{"BatteryVoltage", status.BatteryVoltage, 12.8},
			{"BatteryTemp", status.BatteryTemp, 25.0},
			{"DeviceTemp", status.DeviceTemp, 32.0},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if !floatEqual(tt.got, tt.want) {
					t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
				}
			})
		}

		if status.BatterySOC != 85 {
			t.Errorf("BatterySOC = %v, want 85", status.BatterySOC)
		}

		if status.ChargingStatus != 1 {
			t.Errorf("ChargingStatus = %v, want 1", status.ChargingStatus)
		}

		if status.Timestamp == 0 {
			t.Error("Timestamp should be set")
		}

		if status.CollectionTime <= 0 {
			t.Error("CollectionTime should be positive")
		}
	})

	t.Run("modbus read failure for array voltage", func(t *testing.T) {
		mockClient := &testingpkg.MockModbusClient{
			ReadInputRegistersFunc: func(_ context.Context, address, _ uint16) ([]byte, error) {
				if address == regArrayVoltage {
					return nil, &testingpkg.ModbusTestError{Message: "timeout"}
				}
				return testingpkg.CreateModbusResponse(0), nil
			},
		}

		mockMetrics := &testingpkg.MockMetricsCollector{}
		collector := NewCollector(mockClient, mockMetrics)
		_, err := collector.GetStatus(ctx)

		if err == nil {
			t.Error("GetStatus() should return error when modbus read fails")
		}
	})

	t.Run("modbus read failure for batch data", func(t *testing.T) {
		mockClient := &testingpkg.MockModbusClient{
			ReadInputRegistersFunc: func(_ context.Context, address, _ uint16) ([]byte, error) {
				if address == regArrayVoltage {
					return nil, &testingpkg.ModbusTestError{Message: "device disconnected"}
				}
				return testingpkg.CreateModbusResponse(0), nil
			},
		}

		mockMetrics := &testingpkg.MockMetricsCollector{}
		collector := NewCollector(mockClient, mockMetrics)
		_, err := collector.GetStatus(ctx)

		if err == nil {
			t.Error("GetStatus() should return error when batch read fails")
		}
	})

	t.Run("negative temperature handling", func(t *testing.T) {
		mockClient := &testingpkg.MockModbusClient{
			ReadInputRegistersFunc: func(_ context.Context, address, quantity uint16) ([]byte, error) {
				switch address {
				case regArrayVoltage: // Batch read with negative temperatures
					if quantity == 18 {
						// Same as successful test but with negative temperatures
						return testingpkg.CreateModbusResponse(
							1850, 520, // Array V/I
							0, 962, // Array power (32-bit)
							1280,   // Battery voltage
							480,    // Charging current
							0, 614, // Charging power (32-bit)
							0, 0, 0, 0, 0, 0, 0, 0, // Unused registers
							64536, 65036, // Battery temp (-10°C), Device temp (-5°C)
						), nil
					}
					return testingpkg.CreateModbusResponse(1850, 520), nil
				case regBatterySOC:
					return testingpkg.CreateModbusResponse(85), nil
				case regEnergyGeneratedDaily:
					return testingpkg.CreateModbusResponse(0, 1550), nil
				case regControllerStatus:
					return testingpkg.CreateModbusResponse(0x0004), nil
				default:
					return testingpkg.CreateModbusResponse(0), nil
				}
			},
		}

		mockMetrics := &testingpkg.MockMetricsCollector{}
		collector := NewCollector(mockClient, mockMetrics)
		status, err := collector.GetStatus(ctx)

		if err != nil {
			t.Fatalf("GetStatus() error = %v", err)
		}

		if !floatEqual(status.BatteryTemp, -10.0) {
			t.Errorf("BatteryTemp = %v, want -10.0", status.BatteryTemp)
		}

		if !floatEqual(status.DeviceTemp, -5.0) {
			t.Errorf("DeviceTemp = %v, want -5.0", status.DeviceTemp)
		}
	})

	t.Run("insufficient data from modbus", func(t *testing.T) {
		mockClient := &testingpkg.MockModbusClient{
			ReadInputRegistersFunc: func(_ context.Context, _, _ uint16) ([]byte, error) {
				// Return insufficient data (1 byte instead of expected 2+ bytes)
				return []byte{0x00}, nil
			},
		}

		mockMetrics := &testingpkg.MockMetricsCollector{}
		collector := NewCollector(mockClient, mockMetrics)
		_, err := collector.GetStatus(ctx)

		if err == nil {
			t.Error("GetStatus() should return error when modbus returns insufficient data")
		}
	})
}

// floatEqual checks if two float32 values are approximately equal
func floatEqual(a, b float32) bool {
	tolerance := float32(0.01)
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < tolerance
}
