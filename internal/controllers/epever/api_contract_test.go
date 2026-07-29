package epever

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lumberbarons/solar-controller/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The GET handlers build their responses from gin.H literals, so a rename there
// is invisible to the Go compiler and to any test that only checks values. These
// tests pin the field names against docs/api-contract.json, which the frontend
// checks from the other side, so the two cannot drift apart silently.
func TestGetHandlersMatchAPIContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configurer := NewConfigurer(fullConfigMockClient(), &MockMetricsCollector{})

	router := gin.New()
	router.GET("/api/epever/battery-profile", configurer.BatteryProfileGet())
	router.GET("/api/epever/charging-parameters", configurer.ChargingParametersGet())
	router.GET("/api/epever/time", configurer.TimeGet())

	tests := []struct {
		endpoint string
		path     string
	}{
		{endpoint: "GET /api/epever/battery-profile", path: "/api/epever/battery-profile"},
		{endpoint: "GET /api/epever/charging-parameters", path: "/api/epever/charging-parameters"},
		{endpoint: "GET /api/epever/time", path: "/api/epever/time"},
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tt.path, nil))
			require.Equal(t, http.StatusOK, recorder.Code, "body: %s", recorder.Body)

			assert.Equal(t,
				testutil.APIContractFields(t, tt.endpoint),
				testutil.JSONFieldNames(t, recorder.Body.Bytes()),
				"%s no longer returns the fields docs/api-contract.json records; "+
					"update the contract and site/src/api/types.ts together", tt.endpoint)
		})
	}
}

// MetricsGet serialises *ControllerStatus directly, so the struct's json tags
// are the wire contract for that endpoint.
func TestMetricsPayloadMatchesAPIContract(t *testing.T) {
	payload, err := json.Marshal(ControllerStatus{})
	require.NoError(t, err)

	assert.Equal(t,
		testutil.APIContractFields(t, "GET /api/epever/metrics"),
		testutil.JSONFieldNames(t, payload),
		"ControllerStatus no longer marshals to the fields docs/api-contract.json "+
			"records; update the contract and site/src/api/types.ts together")
}
