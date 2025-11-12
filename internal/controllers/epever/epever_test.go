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

		controller, err := NewController(
			mockClient,
			collector,
			configurer,
			mockPublisher,
			mockMetrics,
			"test-device-1",
			60,
		)
		if err != nil {
			t.Fatalf("NewController() error = %v", err)
		}
		defer controller.Close()

		// Wait for the initial collection to complete (triggered in NewController)
		// The collectAndPublish runs asynchronously, so we need to give it time
		// In a real test, we'd use a more sophisticated synchronization mechanism
		// but for now we'll just check the state after the goroutine runs

		// Manually trigger collection to ensure it runs synchronously in test
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
		err = json.Unmarshal([]byte(call.Payload), &payload)
		if err != nil {
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
							0, 962, // Array power (32-bit)
							1280,   // Battery voltage
							480,    // Charging current
							0, 614, // Charging power (32-bit)
							0, 0, 0, 0, 0, 0, 0, 0, // Unused registers
							2500, 3200, // Battery temp, Device temp
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
		mockPublisher := &testingpkg.MockMessagePublisher{}

		collector := NewCollector(mockClient, mockMetrics)
		configurer := NewConfigurer(mockClient, mockMetrics)

		controller, err := NewController(
			mockClient,
			collector,
			configurer,
			mockPublisher,
			mockMetrics,
			"test-device-1",
			60,
		)
		if err != nil {
			t.Fatalf("NewController() error = %v", err)
		}
		defer controller.Close()

		// Manually trigger collection
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
	})
}
