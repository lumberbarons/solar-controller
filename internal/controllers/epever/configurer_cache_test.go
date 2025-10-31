package epever

import (
	"context"
	"testing"
	"time"

	testingpkg "github.com/lumberbarons/solar-controller/internal/controllers/testing"
)

func TestConfigurer_getCachedConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("cache miss - fetches from device", func(t *testing.T) {
		callCount := 0
		mockClient := &testingpkg.MockModbusClient{
			ReadHoldingRegistersFunc: func(_ context.Context, address, _ uint16) ([]byte, error) {
				callCount++
				// Return minimal valid data for getConfig()
				switch address {
				case regBatteryType: // Battery type, capacity, temp comp (3 registers)
					return testingpkg.CreateModbusResponse(4, 100, 3), nil
				case regRealTimeClock: // Time (3 registers, 6 bytes)
					return []byte{0, 0, 1, 12, 25, 1}, nil // HH:MM, Month, Hour, Year-2000, Day
				case regOverVoltDisconnect: // Voltage parameters (12 registers)
					return testingpkg.CreateModbusResponse(
						1600, 1500, 1500, 1460, 1440, 1380,
						1320, 1260, 1220, 1200, 1110, 1080,
					), nil
				case regEqualizationChargingCycle: // Equalization cycle (1 register)
					return testingpkg.CreateModbusResponse(30), nil
				case regEqualizationChargingTime: // Durations (2 registers)
					return testingpkg.CreateModbusResponse(120, 120), nil
				case regBatteryTempUpperLimit: // Temperature limits (4 registers)
					return testingpkg.CreateModbusResponse(4500, 65436, 4500, 4000), nil // 45°C, -10°C, 45°C, 40°C
				case 0x903A: // Power component temps (2 registers)
					return testingpkg.CreateModbusResponse(8500, 8000), nil // 85°C, 80°C
				case 0x903C: // Line impedance (1 register)
					return testingpkg.CreateModbusResponse(0), nil
				case 0x903D: // Night/day threshold (2 registers)
					return testingpkg.CreateModbusResponse(500, 10), nil
				case 0x903F: // Day threshold and delay (2 registers)
					return testingpkg.CreateModbusResponse(600, 10), nil
				default:
					return testingpkg.CreateModbusResponse(0), nil
				}
			},
		}

		mockMetrics := &testingpkg.MockMetricsCollector{}
		configurer := NewConfigurer(mockClient, mockMetrics)

		// First call should fetch from device
		config1, err := configurer.getCachedConfig(ctx)
		if err != nil {
			t.Fatalf("getCachedConfig() error = %v", err)
		}
		if config1 == nil {
			t.Fatal("getCachedConfig() returned nil config")
		}

		if callCount == 0 {
			t.Error("Expected modbus calls to fetch config, got 0")
		}

		// Verify config values
		if config1.BatteryType != "userDefined" {
			t.Errorf("BatteryType = %v, want userDefined", config1.BatteryType)
		}
		if config1.BatteryCapacity != 100 {
			t.Errorf("BatteryCapacity = %v, want 100", config1.BatteryCapacity)
		}
	})

	t.Run("cache hit - no device fetch", func(t *testing.T) {
		callCount := 0
		mockClient := &testingpkg.MockModbusClient{
			ReadHoldingRegistersFunc: func(_ context.Context, address, _ uint16) ([]byte, error) {
				callCount++
				switch address {
				case regBatteryType:
					return testingpkg.CreateModbusResponse(4, 100, 3), nil
				case regRealTimeClock:
					return []byte{0, 0, 1, 12, 25, 1}, nil
				case regOverVoltDisconnect:
					return testingpkg.CreateModbusResponse(
						1600, 1500, 1500, 1460, 1440, 1380,
						1320, 1260, 1220, 1200, 1110, 1080,
					), nil
				case regEqualizationChargingCycle:
					return testingpkg.CreateModbusResponse(30), nil
				case regEqualizationChargingTime:
					return testingpkg.CreateModbusResponse(120, 120), nil
				case regBatteryTempUpperLimit:
					return testingpkg.CreateModbusResponse(4500, 65436, 4500, 4000), nil
				case 0x903A:
					return testingpkg.CreateModbusResponse(8500, 8000), nil
				case 0x903C:
					return testingpkg.CreateModbusResponse(0), nil
				case 0x903D:
					return testingpkg.CreateModbusResponse(500, 10), nil
				case 0x903F:
					return testingpkg.CreateModbusResponse(600, 10), nil
				default:
					return testingpkg.CreateModbusResponse(0), nil
				}
			},
		}

		mockMetrics := &testingpkg.MockMetricsCollector{}
		configurer := NewConfigurer(mockClient, mockMetrics)

		// First call - populates cache
		config1, err := configurer.getCachedConfig(ctx)
		if err != nil {
			t.Fatalf("getCachedConfig() first call error = %v", err)
		}

		firstCallCount := callCount
		if firstCallCount == 0 {
			t.Error("Expected modbus calls on first fetch")
		}

		// Second call - should use cache
		config2, err := configurer.getCachedConfig(ctx)
		if err != nil {
			t.Fatalf("getCachedConfig() second call error = %v", err)
		}

		if callCount != firstCallCount {
			t.Errorf("Expected no additional modbus calls (cache hit), but got %d total calls vs %d on first call",
				callCount, firstCallCount)
		}

		// Verify both configs are equal
		if config1.BatteryType != config2.BatteryType {
			t.Error("Cached config should match original")
		}
		if config1.BatteryCapacity != config2.BatteryCapacity {
			t.Error("Cached config should match original")
		}

		// Verify they are different pointers (copies, not same reference)
		if config1 == config2 {
			t.Error("getCachedConfig should return copies, not the same reference")
		}
	})

	t.Run("cache expiration - refetches after TTL", func(t *testing.T) {
		callCount := 0
		mockClient := &testingpkg.MockModbusClient{
			ReadHoldingRegistersFunc: func(_ context.Context, address, _ uint16) ([]byte, error) {
				callCount++
				switch address {
				case regBatteryType:
					return testingpkg.CreateModbusResponse(4, 100, 3), nil
				case regRealTimeClock:
					return []byte{0, 0, 1, 12, 25, 1}, nil
				case regOverVoltDisconnect:
					return testingpkg.CreateModbusResponse(
						1600, 1500, 1500, 1460, 1440, 1380,
						1320, 1260, 1220, 1200, 1110, 1080,
					), nil
				case regEqualizationChargingCycle:
					return testingpkg.CreateModbusResponse(30), nil
				case regEqualizationChargingTime:
					return testingpkg.CreateModbusResponse(120, 120), nil
				case regBatteryTempUpperLimit:
					return testingpkg.CreateModbusResponse(4500, 65436, 4500, 4000), nil
				case 0x903A:
					return testingpkg.CreateModbusResponse(8500, 8000), nil
				case 0x903C:
					return testingpkg.CreateModbusResponse(0), nil
				case 0x903D:
					return testingpkg.CreateModbusResponse(500, 10), nil
				case 0x903F:
					return testingpkg.CreateModbusResponse(600, 10), nil
				default:
					return testingpkg.CreateModbusResponse(0), nil
				}
			},
		}

		mockMetrics := &testingpkg.MockMetricsCollector{}
		configurer := NewConfigurer(mockClient, mockMetrics)
		// Set a very short TTL for testing
		configurer.cacheTTL = 1 * time.Millisecond

		// First call - populates cache
		_, err := configurer.getCachedConfig(ctx)
		if err != nil {
			t.Fatalf("getCachedConfig() first call error = %v", err)
		}

		firstCallCount := callCount

		// Wait for cache to expire
		time.Sleep(10 * time.Millisecond)

		// Second call - cache expired, should refetch
		_, err = configurer.getCachedConfig(ctx)
		if err != nil {
			t.Fatalf("getCachedConfig() second call error = %v", err)
		}

		if callCount == firstCallCount {
			t.Error("Expected cache to expire and refetch from device")
		}
	})

	t.Run("cache invalidation", func(t *testing.T) {
		callCount := 0
		mockClient := &testingpkg.MockModbusClient{
			ReadHoldingRegistersFunc: func(_ context.Context, address, _ uint16) ([]byte, error) {
				callCount++
				switch address {
				case regBatteryType:
					return testingpkg.CreateModbusResponse(4, 100, 3), nil
				case regRealTimeClock:
					return []byte{0, 0, 1, 12, 25, 1}, nil
				case regOverVoltDisconnect:
					return testingpkg.CreateModbusResponse(
						1600, 1500, 1500, 1460, 1440, 1380,
						1320, 1260, 1220, 1200, 1110, 1080,
					), nil
				case regEqualizationChargingCycle:
					return testingpkg.CreateModbusResponse(30), nil
				case regEqualizationChargingTime:
					return testingpkg.CreateModbusResponse(120, 120), nil
				case regBatteryTempUpperLimit:
					return testingpkg.CreateModbusResponse(4500, 65436, 4500, 4000), nil
				case 0x903A:
					return testingpkg.CreateModbusResponse(8500, 8000), nil
				case 0x903C:
					return testingpkg.CreateModbusResponse(0), nil
				case 0x903D:
					return testingpkg.CreateModbusResponse(500, 10), nil
				case 0x903F:
					return testingpkg.CreateModbusResponse(600, 10), nil
				default:
					return testingpkg.CreateModbusResponse(0), nil
				}
			},
		}

		mockMetrics := &testingpkg.MockMetricsCollector{}
		configurer := NewConfigurer(mockClient, mockMetrics)

		// First call - populates cache
		_, err := configurer.getCachedConfig(ctx)
		if err != nil {
			t.Fatalf("getCachedConfig() first call error = %v", err)
		}

		firstCallCount := callCount

		// Invalidate cache
		configurer.invalidateCache()

		// Next call should refetch even though TTL hasn't expired
		_, err = configurer.getCachedConfig(ctx)
		if err != nil {
			t.Fatalf("getCachedConfig() after invalidation error = %v", err)
		}

		if callCount == firstCallCount {
			t.Error("Expected refetch after cache invalidation")
		}
	})

	t.Run("modbus error propagates", func(t *testing.T) {
		mockClient := &testingpkg.MockModbusClient{
			ReadHoldingRegistersFunc: func(_ context.Context, _, _ uint16) ([]byte, error) {
				return nil, &testingpkg.ModbusTestError{Message: "device timeout"}
			},
		}

		mockMetrics := &testingpkg.MockMetricsCollector{}
		configurer := NewConfigurer(mockClient, mockMetrics)

		_, err := configurer.getCachedConfig(ctx)
		if err == nil {
			t.Error("Expected error when modbus fails, got nil")
		}
	})
}

func TestConfigurer_invalidateCache(t *testing.T) {
	mockClient := &testingpkg.MockModbusClient{}
	mockMetrics := &testingpkg.MockMetricsCollector{}
	configurer := NewConfigurer(mockClient, mockMetrics)

	// Manually set cache
	configurer.cache = &cachedConfig{
		config:    &ControllerConfig{BatteryCapacity: 100},
		timestamp: time.Now(),
	}

	// Verify cache exists
	if configurer.cache == nil {
		t.Fatal("Cache should be set")
	}

	// Invalidate
	configurer.invalidateCache()

	// Verify cache is cleared
	configurer.cacheMutex.RLock()
	cacheIsNil := configurer.cache == nil
	configurer.cacheMutex.RUnlock()

	if !cacheIsNil {
		t.Error("Cache should be nil after invalidation")
	}
}
