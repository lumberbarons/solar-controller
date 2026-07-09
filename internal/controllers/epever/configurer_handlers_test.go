package epever

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	testingpkg "github.com/lumberbarons/solar-controller/internal/controllers/testing"
)

// newPatchTestRouter returns a router with all four config PATCH handlers
// registered against a mock modbus client that reports a userDefined battery.
func newPatchTestRouter(mockClient *testingpkg.MockModbusClient) *gin.Engine {
	gin.SetMode(gin.TestMode)
	configurer := NewConfigurer(mockClient, &testingpkg.MockMetricsCollector{})

	router := gin.New()
	router.PATCH("/api/epever/config", configurer.ConfigPatch())
	router.PATCH("/api/epever/battery-profile", configurer.BatteryProfilePatch())
	router.PATCH("/api/epever/charging-parameters", configurer.ChargingParametersPatch())
	router.PATCH("/api/epever/time", configurer.TimePatch())
	return router
}

// fullConfigMockClient serves a complete, valid register map so handlers can
// run their read-modify-write flow; writes are recorded but not applied.
func fullConfigMockClient() *testingpkg.MockModbusClient {
	return &testingpkg.MockModbusClient{
		ReadHoldingRegistersFunc: func(_ context.Context, address, _ uint16) ([]byte, error) {
			switch address {
			case regBatteryType:
				return testingpkg.CreateModbusResponse(batteryTypeUserDefined, 100, 3), nil
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
			default:
				return testingpkg.CreateModbusResponse(0), nil
			}
		},
		WriteMultipleRegistersFunc: func(_ context.Context, _, _ uint16, _ []byte) ([]byte, error) {
			return nil, nil
		},
	}
}

func TestPatchHandlers_RejectInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
	}{
		{
			name: "charging parameters voltage above absolute max",
			path: "/api/epever/charging-parameters",
			body: `{"boostVoltage": 700}`,
		},
		{
			name: "charging parameters voltage that would overflow uint16 centivolts",
			path: "/api/epever/charging-parameters",
			body: `{"boostVoltage": 656.0}`,
		},
		{
			name: "charging parameters voltage below absolute min",
			path: "/api/epever/charging-parameters",
			body: `{"floatVoltage": 0.5}`,
		},
		{
			name: "charging parameters malformed voltage field",
			path: "/api/epever/charging-parameters",
			body: `{"boostVoltage": "not-a-number"}`,
		},
		{
			name: "charging parameters malformed duration field",
			path: "/api/epever/charging-parameters",
			body: `{"boostDuration": "not-a-number"}`,
		},
		{
			name: "charging parameters negative duration",
			path: "/api/epever/charging-parameters",
			body: `{"boostDuration": -5}`,
		},
		{
			name: "charging parameters duration above max",
			path: "/api/epever/charging-parameters",
			body: `{"boostDuration": 700}`,
		},
		{
			name: "battery profile unknown battery type",
			path: "/api/epever/battery-profile",
			body: `{"batteryType": "plutonium"}`,
		},
		{
			name: "battery profile malformed capacity",
			path: "/api/epever/battery-profile",
			body: `{"batteryCapacity": "lots"}`,
		},
		{
			name: "battery profile capacity above max",
			path: "/api/epever/battery-profile",
			body: `{"batteryCapacity": 20000}`,
		},
		{
			name: "battery profile zero capacity",
			path: "/api/epever/battery-profile",
			body: `{"batteryCapacity": 0}`,
		},
		{
			name: "battery profile temp comp coefficient above max",
			path: "/api/epever/battery-profile",
			body: `{"tempCompCoefficient": 50}`,
		},
		{
			name: "battery profile temp comp coefficient negative",
			path: "/api/epever/battery-profile",
			body: `{"tempCompCoefficient": -1}`,
		},
		{
			name: "legacy config unknown battery type",
			path: "/api/epever/config",
			body: `{"batteryType": "plutonium"}`,
		},
		{
			name: "legacy config voltage above absolute max",
			path: "/api/epever/config",
			body: `{"boostVoltage": 700}`,
		},
		{
			name: "legacy config duration above max",
			path: "/api/epever/config",
			body: `{"boostDuration": 700}`,
		},
		{
			name: "time year beyond single byte RTC range",
			path: "/api/epever/time",
			body: `{"time": "2300-01-01T00:00:00Z"}`,
		},
		{
			name: "time year before RTC epoch",
			path: "/api/epever/time",
			body: `{"time": "1999-12-31T00:00:00Z"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := fullConfigMockClient()
			router := newPatchTestRouter(mockClient)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("PATCH", tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d (body: %s)", w.Code, http.StatusBadRequest, w.Body.String())
			}
			if n := len(mockClient.WriteMultipleRegistersCalls); n != 0 {
				t.Errorf("invalid request triggered %d modbus writes, want 0", n)
			}
		})
	}
}

func TestPatchHandlers_AcceptValidValues(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
	}{
		{
			name: "battery profile valid capacity",
			path: "/api/epever/battery-profile",
			body: `{"batteryCapacity": 200}`,
		},
		{
			name: "charging parameters valid boost voltage",
			path: "/api/epever/charging-parameters",
			body: `{"boostVoltage": 14.5}`,
		},
		{
			name: "time within RTC range",
			path: "/api/epever/time",
			body: `{"time": "2026-07-09T12:00:00Z"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := fullConfigMockClient()
			router := newPatchTestRouter(mockClient)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("PATCH", tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("status = %d, want %d (body: %s)", w.Code, http.StatusOK, w.Body.String())
			}
			if n := len(mockClient.WriteMultipleRegistersCalls); n == 0 {
				t.Error("valid request performed no modbus writes, want at least 1")
			}
		})
	}
}

func TestPatchHandlers_RejectOversizedBody(t *testing.T) {
	paths := []string{
		"/api/epever/config",
		"/api/epever/battery-profile",
		"/api/epever/charging-parameters",
		"/api/epever/time",
	}

	// Valid JSON comfortably above the body cap
	oversized := `{"padding":"` + strings.Repeat("x", maxPatchBodyBytes+1024) + `"}`

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			mockClient := &testingpkg.MockModbusClient{
				ReadHoldingRegistersFunc: func(_ context.Context, _, _ uint16) ([]byte, error) {
					// Battery type userDefined so handlers proceed to binding
					return testingpkg.CreateModbusResponse(batteryTypeUserDefined), nil
				},
			}
			router := newPatchTestRouter(mockClient)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("PATCH", path, strings.NewReader(oversized))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusRequestEntityTooLarge {
				t.Errorf("status = %d, want %d", w.Code, http.StatusRequestEntityTooLarge)
			}
			if n := len(mockClient.WriteMultipleRegistersCalls); n != 0 {
				t.Errorf("oversized body triggered %d modbus writes, want 0", n)
			}
		})
	}
}
