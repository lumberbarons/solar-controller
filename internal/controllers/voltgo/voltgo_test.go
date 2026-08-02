package voltgo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lumberbarons/solar-controller/internal/publish"
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

// newTestController wires a controller around one battery, matching the
// single-battery shape most behavioural assertions only need.
func newTestController(
	collector *Collector,
	publisher publish.MessagePublisher,
	metrics MetricsCollector,
	deviceID string,
) (*Controller, *batteryRunner) {
	runner := newBatteryRunner("bank-a", "AA:BB:CC:DD:EE:FF", collector, publisher, metrics, deviceID)
	return newControllerForTest([]*batteryRunner{runner}, &MockBatteryConnector{}), runner
}

func TestController_CollectAndPublish(t *testing.T) {
	t.Run("publishes failure metric when collection fails", func(t *testing.T) {
		mockMetrics := &MockMetricsCollector{}
		mockPublisher := &testutil.MockMessagePublisher{}

		controller, _ := newTestController(newFailingCollector(), mockPublisher, mockMetrics, "test-device-1")
		controller.collectAndPublish()

		if mockMetrics.FailuresCount != 1 {
			t.Errorf("Expected FailuresCount = 1, got %d", mockMetrics.FailuresCount)
		}
		if len(mockMetrics.FailureIDs) != 1 || mockMetrics.FailureIDs[0] != "bank-a" {
			t.Errorf("failure was not attributed to a battery: %v", mockMetrics.FailureIDs)
		}

		if len(mockPublisher.PublishCalls) != 1 {
			t.Fatalf("Expected 1 publish call, got %d", len(mockPublisher.PublishCalls))
		}

		call := mockPublisher.PublishCalls[0]
		expectedTopicSuffix := "test-device-1/voltgo/bank-a/collection-failure"
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

		controller, runner := newTestController(collector, mockPublisher, mockMetrics, "test-device-1")
		controller.collectAndPublish()

		if mockMetrics.FailuresCount != 0 {
			t.Errorf("Expected FailuresCount = 0, got %d", mockMetrics.FailuresCount)
		}

		if len(mockMetrics.SetMetricsCalls) != 1 {
			t.Fatalf("Expected 1 SetMetrics call, got %d", len(mockMetrics.SetMetricsCalls))
		}
		if got := mockMetrics.SetMetricsCalls[0].BatteryID; got != "bank-a" {
			t.Errorf("SetMetrics battery id = %q, want %q", got, "bank-a")
		}
		if got := mockMetrics.SetMetricsCalls[0].Status.Voltage; got != 13.28 {
			t.Errorf("SetMetrics status voltage = %v, want 13.28", got)
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
			expectedSuffix := "test-device-1/voltgo/bank-a/" + expectedMetric
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
			suffix := "test-device-1/voltgo/bank-a/" + pc.metric
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

		if runner.status() == nil {
			t.Error("lastStatus should be cached after successful collection")
		}
	})

	t.Run("fetches battery info once and caches it", func(t *testing.T) {
		collector, mockBattery, _ := newWorkingCollector()
		controller, runner := newTestController(collector, &testutil.MockMessagePublisher{}, &MockMetricsCollector{}, "test-device-1")

		controller.collectAndPublish()
		controller.collectAndPublish()

		if mockBattery.GetInfoCalls != 1 {
			t.Errorf("GetInfo calls = %d, want 1 (info should be cached after first fetch)", mockBattery.GetInfoCalls)
		}

		info := runner.info()
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
		controller, runner := newTestController(collector, mockPublisher, mockMetrics, "test-device-1")

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

		if runner.info() == nil {
			t.Error("lastInfo should be cached after retry")
		}
	})
}

// newMultiBatteryController wires three batteries onto one controller. The
// middle one is unreachable, which is the case the deployment actually hits:
// four packs on one BLE bus, of which some are not advertising.
func newMultiBatteryController(
	publisher publish.MessagePublisher,
	metrics MetricsCollector,
) (*Controller, map[string]*MockBatteryClient) {
	clients := make(map[string]*MockBatteryClient, 2)

	newRunner := func(id, address string, fail bool) *batteryRunner {
		if fail {
			connector := &MockBatteryConnector{
				ConnectFunc: func(_ context.Context, addr string) (BatteryClient, error) {
					return nil, errors.New("no advertisement from " + addr)
				},
			}
			collector := NewCollector(connector, address, time.Second)
			return newBatteryRunner(id, address, collector, publisher, metrics, "test-device-1")
		}

		client := &MockBatteryClient{
			GetStatusFunc: func(_ context.Context) (*battery.Status, error) {
				return testBatteryStatus(), nil
			},
			GetInfoFunc: func(_ context.Context) (*battery.Info, error) {
				return &battery.Info{Chemistry: "LiFePO4", NominalVoltage: 12.8, CapacityAh: 100}, nil
			},
		}
		clients[id] = client
		connector := &MockBatteryConnector{
			ConnectFunc: func(_ context.Context, _ string) (BatteryClient, error) {
				return client, nil
			},
		}
		collector := NewCollector(connector, address, time.Second)
		return newBatteryRunner(id, address, collector, publisher, metrics, "test-device-1")
	}

	runners := []*batteryRunner{
		newRunner("bank-a", "AA:AA:AA:AA:AA:01", false),
		newRunner("bank-b", "AA:AA:AA:AA:AA:02", true),
		newRunner("bank-c", "AA:AA:AA:AA:AA:03", false),
	}

	return newControllerForTest(runners, &MockBatteryConnector{}), clients
}

func TestController_MultipleBatteries(t *testing.T) {
	t.Run("every battery is collected and published under its own id", func(t *testing.T) {
		mockMetrics := &MockMetricsCollector{}
		mockPublisher := &testutil.MockMessagePublisher{}

		controller, clients := newMultiBatteryController(mockPublisher, mockMetrics)
		controller.collectAndPublish()

		for _, id := range []string{"bank-a", "bank-c"} {
			if clients[id].GetStatusCalls != 1 {
				t.Errorf("battery %s GetStatus calls = %d, want 1", id, clients[id].GetStatusCalls)
			}
		}

		// Eight metrics each for the two reachable batteries, one failure
		// metric for the third.
		if len(mockPublisher.PublishCalls) != 17 {
			t.Fatalf("publish calls = %d, want 17", len(mockPublisher.PublishCalls))
		}

		perBattery := map[string]int{}
		for _, call := range mockPublisher.PublishCalls {
			parts := strings.Split(call.TopicSuffix, "/")
			if len(parts) != 4 {
				t.Fatalf("topic %q does not have the {device}/voltgo/{battery}/{metric} shape", call.TopicSuffix)
			}
			if parts[1] != "voltgo" {
				t.Errorf("topic %q: segment 2 = %q, want voltgo", call.TopicSuffix, parts[1])
			}
			perBattery[parts[2]]++
		}

		want := map[string]int{"bank-a": 8, "bank-b": 1, "bank-c": 8}
		for id, wantCount := range want {
			if perBattery[id] != wantCount {
				t.Errorf("battery %s published %d metrics, want %d", id, perBattery[id], wantCount)
			}
		}
	})

	t.Run("one battery failing does not stop the others", func(t *testing.T) {
		mockMetrics := &MockMetricsCollector{}
		mockPublisher := &testutil.MockMessagePublisher{}

		controller, _ := newMultiBatteryController(mockPublisher, mockMetrics)
		controller.collectAndPublish()

		if mockMetrics.FailuresCount != 1 {
			t.Errorf("FailuresCount = %d, want 1", mockMetrics.FailuresCount)
		}
		if len(mockMetrics.FailureIDs) != 1 || mockMetrics.FailureIDs[0] != "bank-b" {
			t.Errorf("failure ids = %v, want [bank-b]", mockMetrics.FailureIDs)
		}

		// bank-c is collected after the battery that fails, so its presence
		// here is what proves the cycle carried on past the failure.
		got := mockMetrics.metricsBatteryIDs()
		want := []string{"bank-a", "bank-c"}
		if len(got) != len(want) {
			t.Fatalf("SetMetrics battery ids = %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("SetMetrics battery ids = %v, want %v", got, want)
			}
		}
	})

	t.Run("each battery caches its own status independently", func(t *testing.T) {
		controller, _ := newMultiBatteryController(&testutil.MockMessagePublisher{}, &MockMetricsCollector{})
		controller.collectAndPublish()

		if controller.byID["bank-a"].status() == nil {
			t.Error("bank-a should have a cached status")
		}
		if controller.byID["bank-b"].status() != nil {
			t.Error("bank-b never collected successfully, so it should have no cached status")
		}
		if controller.byID["bank-c"].status() == nil {
			t.Error("bank-c should have a cached status")
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

	get := func(router *gin.Engine, path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		return w
	}

	t.Run("metrics and info return 204 before first collection", func(t *testing.T) {
		collector, _, _ := newWorkingCollector()
		controller, _ := newTestController(collector, &testutil.MockMessagePublisher{}, &MockMetricsCollector{}, "test-device-1")
		router := newRouter(controller)

		for _, path := range []string{"/api/voltgo/bank-a/metrics", "/api/voltgo/bank-a/info"} {
			if w := get(router, path); w.Code != http.StatusNoContent {
				t.Errorf("GET %s status = %d, want %d", path, w.Code, http.StatusNoContent)
			}
		}
	})

	t.Run("metrics returns last status including cells", func(t *testing.T) {
		collector, _, _ := newWorkingCollector()
		controller, _ := newTestController(collector, &testutil.MockMessagePublisher{}, &MockMetricsCollector{}, "test-device-1")
		controller.collectAndPublish()
		router := newRouter(controller)

		w := get(router, "/api/voltgo/bank-a/metrics")
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
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
		controller, _ := newTestController(collector, &testutil.MockMessagePublisher{}, &MockMetricsCollector{}, "test-device-1")
		controller.collectAndPublish()
		router := newRouter(controller)

		w := get(router, "/api/voltgo/bank-a/info")
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
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

	t.Run("index lists every configured battery", func(t *testing.T) {
		controller, _ := newMultiBatteryController(&testutil.MockMessagePublisher{}, &MockMetricsCollector{})
		router := newRouter(controller)

		w := get(router, "/api/voltgo")
		if w.Code != http.StatusOK {
			t.Fatalf("GET /api/voltgo status = %d, want %d", w.Code, http.StatusOK)
		}

		var index BatteryIndex
		if err := json.Unmarshal(w.Body.Bytes(), &index); err != nil {
			t.Fatalf("failed to unmarshal index: %v", err)
		}

		ids := make([]string, 0, len(index.Batteries))
		addressByID := map[string]string{}
		for _, ref := range index.Batteries {
			ids = append(ids, ref.ID)
			addressByID[ref.ID] = ref.Address
		}
		sort.Strings(ids)

		want := []string{"bank-a", "bank-b", "bank-c"}
		if len(ids) != len(want) {
			t.Fatalf("index ids = %v, want %v", ids, want)
		}
		for i := range want {
			if ids[i] != want[i] {
				t.Fatalf("index ids = %v, want %v", ids, want)
			}
		}

		// The unreachable battery must still be listed: the panel for it is
		// what shows an operator that it exists and is not reporting.
		if addressByID["bank-b"] != "AA:AA:AA:AA:AA:02" {
			t.Errorf("bank-b address = %q, want AA:AA:AA:AA:AA:02", addressByID["bank-b"])
		}
	})

	t.Run("unknown battery id returns 404", func(t *testing.T) {
		collector, _, _ := newWorkingCollector()
		controller, _ := newTestController(collector, &testutil.MockMessagePublisher{}, &MockMetricsCollector{}, "test-device-1")
		router := newRouter(controller)

		for _, path := range []string{"/api/voltgo/nope/metrics", "/api/voltgo/nope/info"} {
			if w := get(router, path); w.Code != http.StatusNotFound {
				t.Errorf("GET %s status = %d, want %d", path, w.Code, http.StatusNotFound)
			}
		}
	})

	t.Run("disabled controller registers no endpoints", func(t *testing.T) {
		router := newRouter(&Controller{})

		for _, path := range []string{"/api/voltgo", "/api/voltgo/bank-a/metrics"} {
			if w := get(router, path); w.Code != http.StatusNotFound {
				t.Errorf("GET %s on disabled controller status = %d, want %d", path, w.Code, http.StatusNotFound)
			}
		}
	})
}

func TestController_Enabled(t *testing.T) {
	disabled := &Controller{}
	if disabled.Enabled() {
		t.Error("empty controller should not be enabled")
	}

	collector, _, _ := newWorkingCollector()
	enabled, _ := newTestController(collector, &testutil.MockMessagePublisher{}, &MockMetricsCollector{}, "test-device-1")
	if !enabled.Enabled() {
		t.Error("controller with a battery should be enabled")
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
		collector, mockBattery, _ := newWorkingCollector()
		sharedConnector := &MockBatteryConnector{}
		runner := newBatteryRunner("bank-a", "AA:BB:CC:DD:EE:FF", collector,
			&testutil.MockMessagePublisher{}, &MockMetricsCollector{}, "test-device-1")
		controller := newControllerForTest([]*batteryRunner{runner}, sharedConnector)
		controller.collectAndPublish()

		if err := controller.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		if mockBattery.DisconnectCalls != 1 {
			t.Errorf("Disconnect calls = %d, want 1", mockBattery.DisconnectCalls)
		}
		if sharedConnector.CloseCalls != 1 {
			t.Errorf("connector Close calls = %d, want 1", sharedConnector.CloseCalls)
		}
	})

	// The BLE adapter is shared across every battery, so closing the
	// controller must release it once - not once per battery, which would
	// double-free the adapter.
	t.Run("shared adapter is released exactly once for several batteries", func(t *testing.T) {
		sharedConnector := &MockBatteryConnector{
			ConnectFunc: func(_ context.Context, _ string) (BatteryClient, error) {
				return &MockBatteryClient{}, nil
			},
		}

		runners := make([]*batteryRunner, 0, 3)
		for _, id := range []string{"bank-a", "bank-b", "bank-c"} {
			collector := NewCollector(sharedConnector, "AA:AA:AA:AA:AA:01", time.Second)
			runners = append(runners, newBatteryRunner(id, "AA:AA:AA:AA:AA:01", collector,
				&testutil.MockMessagePublisher{}, &MockMetricsCollector{}, "test-device-1"))
		}

		controller := newControllerForTest(runners, sharedConnector)
		if err := controller.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		if sharedConnector.CloseCalls != 1 {
			t.Errorf("connector Close calls = %d, want 1", sharedConnector.CloseCalls)
		}
	})
}

func TestNewControllerFromConfig_Disabled(t *testing.T) {
	tests := []struct {
		name   string
		config Configuration
	}{
		{"disabled via configuration", Configuration{Enabled: false}},
		{"enabled but no battery configured", Configuration{Enabled: true}},
		{
			name: "enabled but a battery has no address",
			config: Configuration{
				Enabled:   true,
				Batteries: []BatteryConfiguration{{ID: "bank-a"}},
			},
		},
		{
			name: "enabled but two batteries share an id",
			config: Configuration{
				Enabled: true,
				Batteries: []BatteryConfiguration{
					{ID: "bank-a", Address: "AA:AA:AA:AA:AA:01"},
					{ID: "bank-a", Address: "AA:AA:AA:AA:AA:02"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, err := NewControllerFromConfig(tt.config, &testutil.MockMessagePublisher{}, "test-device-1")
			if err != nil {
				t.Fatalf("NewControllerFromConfig() error = %v", err)
			}
			if controller.Enabled() {
				t.Error("controller should be disabled")
			}
		})
	}
}
