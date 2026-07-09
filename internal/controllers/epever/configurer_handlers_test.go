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
