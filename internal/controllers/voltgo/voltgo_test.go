package voltgo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lumberbarons/solar-controller/internal/testutil"
	"github.com/lumberbarons/voltgo/battery"
)

func newWorkingCollector() (*Collector, *MockBatteryClient, *MockBatteryConnector) {
	mockBattery := &MockBatteryClient{
		GetStatusFunc: func(_ context.Context) (*battery.Status, error) {
			return testBatteryStatus(), nil
		},
		GetInfoFunc: func(_ context.Context) (*battery.Info, error) {
			return &battery.Info{
				Chemistry:      "LiFePO4",
				NominalVoltage: 12.8,
				CapacityAh:     100,
				DeviceStrings:  []string{"VOLTGO-100"},
			}, nil
		},
	}
	mockConnector := &MockBatteryConnector{
		ConnectFunc: func(_ context.Context, _ string) (BatteryClient, error) {
			return mockBattery, nil
		},
	}
	return NewCollector(mockConnector, "AA:BB:CC:DD:EE:FF", 10*time.Second), mockBattery, mockConnector
}

func newFailingCollector() *Collector {
	mockConnector := &MockBatteryConnector{
		ConnectFunc: func(_ context.Context, _ string) (BatteryClient, error) {
			return nil, errors.New("device not found")
		},
	}
	return NewCollector(mockConnector, "AA:BB:CC:DD:EE:FF", 10*time.Second)
}

func TestController_CollectAndPublish(t *testing.T) {
	t.Run("publishes failure metric when collection fails", func(t *testing.T) {
		mockMetrics := &MockMetricsCollector{}
		mockPublisher := &testutil.MockMessagePublisher{}

		controller := newControllerForTest(newFailingCollector(), mockPublisher, mockMetrics, "test-device-1")
		controller.collectAndPublish()

		if mockMetrics.FailuresCount != 1 {
			t.Errorf("Expected FailuresCount = 1, got %d", mockMetrics.FailuresCount)
		}

		if len(mockPublisher.PublishCalls) != 1 {
			t.Fatalf("Expected 1 publish call, got %d", len(mockPublisher.PublishCalls))
		}

		call := mockPublisher.PublishCalls[0]
		expectedTopicSuffix := "test-device-1/voltgo/collection-failure"
		if call.TopicSuffix != expectedTopicSuffix {
			t.Errorf("Expected topic suffix %q, got %q", expectedTopicSuffix, call.TopicSuffix)
		}

		var payload MetricPayload
		if err := json.Unmarshal([]byte(call.Payload), &payload); err != nil {
			t.Fatalf("Failed to unmarshal payload: %v", err)
		}
		if payload.Value != float64(1) {
			t.Errorf("Expected failure metric value = 1, got %v", payload.Value)
		}
		if payload.Unit != "count" {
			t.Errorf("Expected failure metric unit = 'count', got %q", payload.Unit)
		}
	})

	t.Run("publishes normal metrics when collection succeeds", func(t *testing.T) {
		collector, _, _ := newWorkingCollector()
		mockMetrics := &MockMetricsCollector{}
		mockPublisher := &testutil.MockMessagePublisher{}

		controller := newControllerForTest(collector, mockPublisher, mockMetrics, "test-device-1")
		controller.collectAndPublish()

		if mockMetrics.FailuresCount != 0 {
			t.Errorf("Expected FailuresCount = 0, got %d", mockMetrics.FailuresCount)
		}

		if len(mockMetrics.SetMetricsCalls) != 1 {
			t.Errorf("Expected 1 SetMetrics call, got %d", len(mockMetrics.SetMetricsCalls))
		}

		if len(mockPublisher.PublishCalls) != 8 {
			t.Fatalf("Expected 8 publish calls for normal metrics, got %d", len(mockPublisher.PublishCalls))
		}

		for _, call := range mockPublisher.PublishCalls {
			if strings.HasSuffix(call.TopicSuffix, "collection-failure") {
				t.Error("collection-failure metric should not be published on successful collection")
			}
		}

		metricNames := []string{
			"battery-voltage", "battery-current", "battery-power",
			"battery-soc", "battery-soh", "battery-temp",
			"cell-voltage-delta", "collection-time",
		}
		for _, expectedMetric := range metricNames {
			found := false
			expectedSuffix := "test-device-1/voltgo/" + expectedMetric
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

		payloadChecks := []struct {
			metric string
			value  float64
			unit   string
		}{
			{"battery-voltage", 13.28, "volts"},
			{"battery-soc", 87, "percent"},
			{"battery-current", -2.5, "amperes"},
		}
		for _, pc := range payloadChecks {
			suffix := "test-device-1/voltgo/" + pc.metric
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
					break
				}
			}
		}

		controller.lastStatusMutex.RLock()
		lastStatus := controller.lastStatus
		controller.lastStatusMutex.RUnlock()
		if lastStatus == nil {
			t.Error("lastStatus should be cached after successful collection")
		}
	})

	t.Run("fetches battery info once and caches it", func(t *testing.T) {
		collector, mockBattery, _ := newWorkingCollector()
		controller := newControllerForTest(collector, &testutil.MockMessagePublisher{}, &MockMetricsCollector{}, "test-device-1")

		controller.collectAndPublish()
		controller.collectAndPublish()

		if mockBattery.GetInfoCalls != 1 {
			t.Errorf("GetInfo calls = %d, want 1 (info should be cached after first fetch)", mockBattery.GetInfoCalls)
		}

		controller.lastStatusMutex.RLock()
		info := controller.lastInfo
		controller.lastStatusMutex.RUnlock()
		if info == nil {
			t.Fatal("lastInfo should be cached")
		}
		if info.Chemistry != "LiFePO4" {
			t.Errorf("Chemistry = %s, want LiFePO4", info.Chemistry)
		}
	})

	t.Run("info fetch failure does not fail the collection cycle", func(t *testing.T) {
		collector, mockBattery, _ := newWorkingCollector()
		mockBattery.GetInfoFunc = func(_ context.Context) (*battery.Info, error) {
			return nil, errors.New("BLE read timeout")
		}

		mockMetrics := &MockMetricsCollector{}
		mockPublisher := &testutil.MockMessagePublisher{}
		controller := newControllerForTest(collector, mockPublisher, mockMetrics, "test-device-1")

		controller.collectAndPublish()

		if mockMetrics.FailuresCount != 0 {
			t.Errorf("Expected FailuresCount = 0, got %d", mockMetrics.FailuresCount)
		}
		if len(mockPublisher.PublishCalls) != 8 {
			t.Errorf("Expected 8 publish calls, got %d", len(mockPublisher.PublishCalls))
		}

		// Info fetch is retried on the next cycle after a failure
		mockBattery.GetInfoFunc = func(_ context.Context) (*battery.Info, error) {
			return &battery.Info{Chemistry: "LiFePO4"}, nil
		}
		controller.collectAndPublish()

		controller.lastStatusMutex.RLock()
		info := controller.lastInfo
		controller.lastStatusMutex.RUnlock()
		if info == nil {
			t.Error("lastInfo should be cached after retry")
		}
	})
}

func TestController_Endpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newRouter := func(controller *Controller) *gin.Engine {
		router := gin.New()
		controller.RegisterEndpoints(router)
		return router
	}

	t.Run("metrics and info return 204 before first collection", func(t *testing.T) {
		collector, _, _ := newWorkingCollector()
		controller := newControllerForTest(collector, &testutil.MockMessagePublisher{}, &MockMetricsCollector{}, "test-device-1")
		router := newRouter(controller)

		for _, path := range []string{"/api/voltgo/metrics", "/api/voltgo/info"} {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			router.ServeHTTP(w, req)
			if w.Code != http.StatusNoContent {
				t.Errorf("GET %s status = %d, want %d", path, w.Code, http.StatusNoContent)
			}
		}
	})

	t.Run("metrics returns last status including cells", func(t *testing.T) {
		collector, _, _ := newWorkingCollector()
		controller := newControllerForTest(collector, &testutil.MockMessagePublisher{}, &MockMetricsCollector{}, "test-device-1")
		controller.collectAndPublish()
		router := newRouter(controller)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/voltgo/metrics", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("GET /api/voltgo/metrics status = %d, want %d", w.Code, http.StatusOK)
		}

		var status BatteryStatus
		if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if status.Voltage != 13.28 {
			t.Errorf("voltage = %v, want 13.28", status.Voltage)
		}
		if len(status.Cells) != 4 {
			t.Errorf("cells length = %d, want 4", len(status.Cells))
		}
	})

	t.Run("info returns cached battery info", func(t *testing.T) {
		collector, _, _ := newWorkingCollector()
		controller := newControllerForTest(collector, &testutil.MockMessagePublisher{}, &MockMetricsCollector{}, "test-device-1")
		controller.collectAndPublish()
		router := newRouter(controller)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/voltgo/info", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("GET /api/voltgo/info status = %d, want %d", w.Code, http.StatusOK)
		}

		var info BatteryInfo
		if err := json.Unmarshal(w.Body.Bytes(), &info); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if info.Chemistry != "LiFePO4" {
			t.Errorf("chemistry = %s, want LiFePO4", info.Chemistry)
		}
		if info.CapacityAh != 100 {
			t.Errorf("capacityAh = %v, want 100", info.CapacityAh)
		}
	})

	t.Run("disabled controller registers no endpoints", func(t *testing.T) {
		controller := &Controller{}
		router := newRouter(controller)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/voltgo/metrics", nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("GET /api/voltgo/metrics on disabled controller status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

func TestController_Enabled(t *testing.T) {
	disabled := &Controller{}
	if disabled.Enabled() {
		t.Error("empty controller should not be enabled")
	}

	collector, _, _ := newWorkingCollector()
	enabled := newControllerForTest(collector, &testutil.MockMessagePublisher{}, &MockMetricsCollector{}, "test-device-1")
	if !enabled.Enabled() {
		t.Error("controller with a collector should be enabled")
	}
}

func TestController_Close(t *testing.T) {
	t.Run("disabled controller close is a no-op", func(t *testing.T) {
		controller := &Controller{}
		if err := controller.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	t.Run("close disconnects battery and releases adapter", func(t *testing.T) {
		collector, mockBattery, mockConnector := newWorkingCollector()
		controller := newControllerForTest(collector, &testutil.MockMessagePublisher{}, &MockMetricsCollector{}, "test-device-1")
		controller.collectAndPublish()

		if err := controller.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		if mockBattery.DisconnectCalls != 1 {
			t.Errorf("Disconnect calls = %d, want 1", mockBattery.DisconnectCalls)
		}
		if mockConnector.CloseCalls != 1 {
			t.Errorf("connector Close calls = %d, want 1", mockConnector.CloseCalls)
		}
	})
}

func TestNewControllerFromConfig_Disabled(t *testing.T) {
	t.Run("disabled via configuration", func(t *testing.T) {
		controller, err := NewControllerFromConfig(Configuration{Enabled: false}, &testutil.MockMessagePublisher{}, "test-device-1")
		if err != nil {
			t.Fatalf("NewControllerFromConfig() error = %v", err)
		}
		if controller.Enabled() {
			t.Error("controller should be disabled")
		}
	})

	t.Run("enabled but missing address", func(t *testing.T) {
		controller, err := NewControllerFromConfig(Configuration{Enabled: true}, &testutil.MockMessagePublisher{}, "test-device-1")
		if err != nil {
			t.Fatalf("NewControllerFromConfig() error = %v", err)
		}
		if controller.Enabled() {
			t.Error("controller should be disabled when address is missing")
		}
	})
}
