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
			ReadInputRegistersFunc: func(_ context.Context, address, _ uint16) ([]byte, error) {
				// Return realistic test data based on the address and quantity
				switch address {
				case regArrayVoltage: // Array voltage and current (2 registers)
					return testingpkg.CreateModbusResponse(1850, 520), nil // 18.5V, 5.2A
				case regArrayPower: // Array power (2 registers for 32-bit)
					return testingpkg.CreateModbusResponse(0, 962), nil // ~96.2W
				case regBatteryVoltage: // Battery voltage (1 register)
					return testingpkg.CreateModbusResponse(1280), nil // 12.8V
				case regChargingCurrent: // Charging current (1 register)
					return testingpkg.CreateModbusResponse(480), nil // 4.8A
				case regChargingPower: // Charging power (2 registers for 32-bit)
					return testingpkg.CreateModbusResponse(0, 614), nil // ~61.4W
				case regBatterySOC: // Battery SOC (1 register)
					return testingpkg.CreateModbusResponse(85), nil // 85%
				case regBatteryTemperature: // Battery and device temp (2 registers)
					return testingpkg.CreateModbusResponse(2500, 3200), nil // 25°C, 32°C
				case regBatteryMaxVoltage: // Max voltage (reading 2: max then min)
					return testingpkg.CreateModbusResponse(1440, 1200), nil // 14.4V, 12.0V
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

		collector := NewCollector(mockClient)
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
			{"BatteryMinVoltage", status.BatteryMinVoltage, 12.0},
			{"BatteryMaxVoltage", status.BatteryMaxVoltage, 14.4},
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

		collector := NewCollector(mockClient)
		_, err := collector.GetStatus(ctx)

		if err == nil {
			t.Error("GetStatus() should return error when modbus read fails")
		}
	})

	t.Run("modbus read failure for temperature", func(t *testing.T) {
		mockClient := &testingpkg.MockModbusClient{
			ReadInputRegistersFunc: func(_ context.Context, address, _ uint16) ([]byte, error) {
				if address == regBatteryTemperature {
					return nil, &testingpkg.ModbusTestError{Message: "device disconnected"}
				}
				// Return valid data for other addresses to get to temperature read
				switch address {
				case regArrayVoltage:
					return testingpkg.CreateModbusResponse(1850, 520), nil
				case regArrayPower:
					return testingpkg.CreateModbusResponse(0, 962), nil
				case regBatteryVoltage:
					return testingpkg.CreateModbusResponse(1280), nil
				case regChargingCurrent:
					return testingpkg.CreateModbusResponse(480), nil
				case regChargingPower:
					return testingpkg.CreateModbusResponse(0, 614), nil
				case regBatterySOC:
					return testingpkg.CreateModbusResponse(85), nil
				case regBatteryMinVoltage:
					return testingpkg.CreateModbusResponse(1200), nil
				case regBatteryMaxVoltage:
					return testingpkg.CreateModbusResponse(1440), nil
				case regEnergyGeneratedDaily:
					return testingpkg.CreateModbusResponse(0, 1550), nil
				case regControllerStatus:
					return testingpkg.CreateModbusResponse(0x0004), nil
				default:
					return testingpkg.CreateModbusResponse(0), nil
				}
			},
		}

		collector := NewCollector(mockClient)
		_, err := collector.GetStatus(ctx)

		if err == nil {
			t.Error("GetStatus() should return error when temperature read fails")
		}
	})

	t.Run("negative temperature handling", func(t *testing.T) {
		mockClient := &testingpkg.MockModbusClient{
			ReadInputRegistersFunc: func(_ context.Context, address, _ uint16) ([]byte, error) {
				switch address {
				case regArrayVoltage:
					return testingpkg.CreateModbusResponse(1850, 520), nil
				case regArrayPower:
					return testingpkg.CreateModbusResponse(0, 962), nil
				case regBatteryVoltage:
					return testingpkg.CreateModbusResponse(1280), nil
				case regChargingCurrent:
					return testingpkg.CreateModbusResponse(480), nil
				case regChargingPower:
					return testingpkg.CreateModbusResponse(0, 614), nil
				case regBatterySOC:
					return testingpkg.CreateModbusResponse(85), nil
				case regBatteryTemperature: // -10°C battery, -5°C device
					return testingpkg.CreateModbusResponse(64536, 65036), nil
				case regBatteryMaxVoltage: // Max and Min voltage
					return testingpkg.CreateModbusResponse(1440, 1200), nil
				case regEnergyGeneratedDaily:
					return testingpkg.CreateModbusResponse(0, 1550), nil
				case regControllerStatus:
					return testingpkg.CreateModbusResponse(0x0004), nil
				default:
					return testingpkg.CreateModbusResponse(0), nil
				}
			},
		}

		collector := NewCollector(mockClient)
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

		collector := NewCollector(mockClient)
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
