package epever

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	testingpkg "github.com/lumberbarons/solar-controller/internal/controllers/testing"
)

func TestController_CollectAndPublish_FailureMetric(t *testing.T) {
	t.Run("publishes failure metric when collection fails", func(t *testing.T) {
		// Create a mock client that always fails
		mockClient := &testingpkg.MockModbusClient{
			ReadInputRegistersFunc: testingpkg.CreateModbusError("connection timeout"),
		}

		mockMetrics := &testingpkg.MockMetricsCollector{}
		mockPublisher := &testingpkg.MockMessagePublisher{}

		collector := NewCollector(mockClient, mockMetrics)
		configurer := NewConfigurer(mockClient, mockMetrics)

		controller := newControllerForTest(
			mockClient,
			collector,
			configurer,
			mockPublisher,
			mockMetrics,
			"test-device-1",
		)

		controller.collectAndPublish()

		// Verify failure counter was incremented
		if mockMetrics.FailuresCount != 1 {
			t.Errorf("Expected FailuresCount = 1, got %d", mockMetrics.FailuresCount)
		}

		// Verify failure metric was published
		if len(mockPublisher.PublishCalls) != 1 {
			t.Fatalf("Expected 1 publish call, got %d", len(mockPublisher.PublishCalls))
		}

		call := mockPublisher.PublishCalls[0]

		// Check topic format: {deviceId}/epever/collection-failure
		expectedTopicSuffix := "test-device-1/epever/collection-failure"
		if call.TopicSuffix != expectedTopicSuffix {
			t.Errorf("Expected topic suffix %q, got %q", expectedTopicSuffix, call.TopicSuffix)
		}

		// Check payload structure
		var payload MetricPayload
		if err := json.Unmarshal([]byte(call.Payload), &payload); err != nil {
			t.Fatalf("Failed to unmarshal payload: %v", err)
		}

		// Verify payload fields
		if payload.Value != float64(1) {
			t.Errorf("Expected failure metric value = 1, got %v", payload.Value)
		}

		if payload.Unit != "count" {
			t.Errorf("Expected failure metric unit = 'count', got %q", payload.Unit)
		}

		if payload.Timestamp == 0 {
			t.Error("Expected non-zero timestamp in failure metric")
		}
	})

	t.Run("publishes normal metrics when collection succeeds", func(t *testing.T) {
		// Create a mock client that succeeds
		mockClient := &testingpkg.MockModbusClient{
			ReadInputRegistersFunc: func(_ context.Context, address, quantity uint16) ([]byte, error) {
				switch address {
				case regArrayVoltage:
					if quantity == 18 {
						return testingpkg.CreateModbusResponse(
							1850, 520, // Array V/I
							962, 0, // Array power (32-bit: low word, high word)
							1280,   // Battery voltage
							480,    // Charging current
							614, 0, // Charging power (32-bit: low word, high word)
							0, 0, 0, 0, 0, 0, 0, 0, // Unused registers
							2500, 3200, // Battery temp, Device temp
						), nil
					}
					return testingpkg.CreateModbusResponse(1850, 520), nil
				case regBatterySOC:
					return testingpkg.CreateModbusResponse(85), nil
				case regEnergyGeneratedDaily:
					return testingpkg.CreateModbusResponse(1550, 0), nil // low word, high word
				case regControllerStatus:
					return testingpkg.CreateModbusResponse(0x0004), nil
				default:
					return testingpkg.CreateModbusResponse(0), nil
				}
			},
		}

		mockMetrics := &testingpkg.MockMetricsCollector{}
		mockPublisher := &testingpkg.MockMessagePublisher{}

		collector := NewCollector(mockClient, mockMetrics)
		configurer := NewConfigurer(mockClient, mockMetrics)

		controller := newControllerForTest(
			mockClient,
			collector,
			configurer,
			mockPublisher,
			mockMetrics,
			"test-device-1",
		)

		controller.collectAndPublish()

		// Verify failure counter was NOT incremented
		if mockMetrics.FailuresCount != 0 {
			t.Errorf("Expected FailuresCount = 0, got %d", mockMetrics.FailuresCount)
		}

		// Verify 12 normal metrics were published (not the failure metric)
		if len(mockPublisher.PublishCalls) != 12 {
			t.Fatalf("Expected 12 publish calls for normal metrics, got %d", len(mockPublisher.PublishCalls))
		}

		// Verify none of the published metrics are the failure metric
		for _, call := range mockPublisher.PublishCalls {
			if strings.HasSuffix(call.TopicSuffix, "collection-failure") {
				t.Error("collection-failure metric should not be published on successful collection")
			}
		}

		// Verify we got the expected metric names
		metricNames := []string{
			"array-voltage", "array-current", "array-power",
			"charging-current", "charging-power",
			"battery-voltage", "battery-soc", "battery-temp",
			"device-temp", "energy-generated-daily",
			"charging-status", "collection-time",
		}

		for _, expectedMetric := range metricNames {
			found := false
			expectedSuffix := "test-device-1/epever/" + expectedMetric
			for _, call := range mockPublisher.PublishCalls {
				if call.TopicSuffix == expectedSuffix {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected metric %q not found in publish calls", expectedMetric)
			}
		}

		// Verify payload values for key metrics
		payloadChecks := []struct {
			metric string
			value  float64
			unit   string
		}{
			{"array-voltage", 18.5, "volts"},
			{"battery-soc", 85, "percent"},
			{"battery-voltage", 12.8, "volts"},
		}

		for _, pc := range payloadChecks {
			suffix := "test-device-1/epever/" + pc.metric
			for _, call := range mockPublisher.PublishCalls {
				if call.TopicSuffix == suffix {
					var payload MetricPayload
					if err := json.Unmarshal([]byte(call.Payload), &payload); err != nil {
						t.Fatalf("Failed to unmarshal %s payload: %v", pc.metric, err)
					}
					if payload.Value != pc.value {
						t.Errorf("%s: value = %v, want %v", pc.metric, payload.Value, pc.value)
					}
					if payload.Unit != pc.unit {
						t.Errorf("%s: unit = %q, want %q", pc.metric, payload.Unit, pc.unit)
					}
					if payload.Timestamp == 0 {
						t.Errorf("%s: timestamp should be non-zero", pc.metric)
					}
					break
				}
			}
		}
	})
}
