package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lumberbarons/solar-controller/internal/config"
	"github.com/lumberbarons/solar-controller/internal/controllers/epever"
	controllertesting "github.com/lumberbarons/solar-controller/internal/controllers/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewApplication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		SolarController: config.SolarControllerConfiguration{
			HTTPPort: 8080,
			Epever: epever.Configuration{
				Enabled: false, // Disabled to avoid needing serial port
			},
		},
	}

	mockPublisher := controllertesting.NewMockPublisher()

	app, err := NewApplication(cfg, mockPublisher)
	require.NoError(t, err)
	require.NotNil(t, app)
	defer app.Close()

	assert.NotNil(t, app.router)
	assert.Equal(t, cfg, app.config)
	assert.Equal(t, mockPublisher, app.mqtt)
}

func TestApplication_MetricsEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		SolarController: config.SolarControllerConfiguration{
			HTTPPort: 8080,
			Epever: epever.Configuration{
				Enabled: false,
			},
		},
	}

	mockPublisher := controllertesting.NewMockPublisher()

	app, err := NewApplication(cfg, mockPublisher)
	require.NoError(t, err)
	defer app.Close()

	// Test that /metrics endpoint is registered
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/metrics", nil)
	app.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "# HELP")
}

func TestApplication_Close(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		SolarController: config.SolarControllerConfiguration{
			HTTPPort: 8080,
			Epever: epever.Configuration{
				Enabled: false,
			},
		},
	}

	mockPublisher := controllertesting.NewMockPublisher()

	app, err := NewApplication(cfg, mockPublisher)
	require.NoError(t, err)

	// Should not error when closing
	err = app.Close()
	assert.NoError(t, err)

	// Verify mock publisher was closed
	assert.True(t, mockPublisher.Closed)
}

func TestApplication_SPAFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		SolarController: config.SolarControllerConfiguration{
			HTTPPort: 8080,
			Epever: epever.Configuration{
				Enabled: false,
			},
		},
	}

	mockPublisher := controllertesting.NewMockPublisher()

	app, err := NewApplication(cfg, mockPublisher)
	require.NoError(t, err)
	defer app.Close()

	tests := []struct {
		name string
		path string
	}{
		{
			name: "root path",
			path: "/",
		},
		{
			name: "config path",
			path: "/config",
		},
		{
			name: "arbitrary SPA route",
			path: "/some/nested/route",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", tt.path, nil)
			app.Router().ServeHTTP(w, req)

			// All SPA routes should return 200 and serve index.html
			assert.Equal(t, http.StatusOK, w.Code)
			// The response should be HTML (index.html), not JSON
			assert.NotContains(t, w.Header().Get("Content-Type"), "application/json")
		})
	}
}

func TestApplication_ControllerRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		epeverEnabled  bool
		expectEndpoint bool
	}{
		{
			name:           "epever disabled",
			epeverEnabled:  false,
			expectEndpoint: false,
		},
		{
			name:           "epever enabled but no serial port",
			epeverEnabled:  true,
			expectEndpoint: false, // Should not register without serial port
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				SolarController: config.SolarControllerConfiguration{
					HTTPPort: 8080,
					Epever: epever.Configuration{
						Enabled: tt.epeverEnabled,
						// No SerialPort specified
					},
				},
			}

			mockPublisher := controllertesting.NewMockPublisher()

			app, err := NewApplication(cfg, mockPublisher)
			require.NoError(t, err)
			defer app.Close()

			// Test that epever endpoint is registered (or not)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/solar/metrics", nil)
			app.Router().ServeHTTP(w, req)

			if tt.expectEndpoint {
				// Endpoint should return JSON metrics
				assert.NotEqual(t, http.StatusNotFound, w.Code)
				assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
			} else {
				// With NoRoute handler, unmatched routes return 200 with index.html (SPA fallback)
				// We verify the endpoint doesn't exist by checking it returns HTML, not JSON
				assert.Equal(t, http.StatusOK, w.Code)
				assert.NotContains(t, w.Header().Get("Content-Type"), "application/json")
			}
		})
	}
}
