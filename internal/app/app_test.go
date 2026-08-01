package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lumberbarons/solar-controller/internal/config"
	"github.com/lumberbarons/solar-controller/internal/controllers/epever"
	"github.com/lumberbarons/solar-controller/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getTestVersionInfo returns version info for testing. No value is a substring
// of another, so an assertion satisfied by the wrong field cannot pass.
func getTestVersionInfo() VersionInfo {
	return VersionInfo{
		Version:   "v-ver",
		BuildTime: "b-time",
		GitCommit: "g-commit",
	}
}

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

	mockPublisher := testutil.NewMockPublisher()

	app, err := NewApplication(cfg, mockPublisher, getTestVersionInfo())
	require.NoError(t, err)
	require.NotNil(t, app)
	defer app.Close()

	assert.NotNil(t, app.router)
	assert.Equal(t, cfg, app.config)
	assert.Equal(t, mockPublisher, app.publisher)
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

	mockPublisher := testutil.NewMockPublisher()

	app, err := NewApplication(cfg, mockPublisher, getTestVersionInfo())
	require.NoError(t, err)
	defer app.Close()

	// Test that /metrics endpoint is registered
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/metrics", nil)
	app.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "# HELP")
}

func TestApplication_InfoEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		SolarController: config.SolarControllerConfiguration{
			HTTPPort: 8080,
			Epever: epever.Configuration{
				Enabled: false,
			},
		},
	}

	mockPublisher := testutil.NewMockPublisher()
	versionInfo := getTestVersionInfo()

	app, err := NewApplication(cfg, mockPublisher, versionInfo)
	require.NoError(t, err)
	defer app.Close()

	// Test that /api/info endpoint returns version information
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/info", nil)
	app.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	// Decode into a map rather than VersionInfo: unmarshalling into the struct
	// would rename the keys symmetrically with the handler, so a renamed json
	// tag would still round-trip. Comparing the decoded map to an expected map
	// fails on a wrong value, a missing field, or a renamed key.
	var got map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))

	assert.Equal(t, map[string]string{
		"version":   versionInfo.Version,
		"buildTime": versionInfo.BuildTime,
		"gitCommit": versionInfo.GitCommit,
	}, got)
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

	mockPublisher := testutil.NewMockPublisher()

	app, err := NewApplication(cfg, mockPublisher, getTestVersionInfo())
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

	mockPublisher := testutil.NewMockPublisher()

	app, err := NewApplication(cfg, mockPublisher, getTestVersionInfo())
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
			// The response body should contain HTML content
			assert.Contains(t, w.Body.String(), "<!doctype html>", "response body should contain HTML")
		})
	}
}

// The frontend hides a controller's panel when its endpoint 404s, so an
// unmatched /api route must not be swallowed by the SPA fallback.
func TestApplication_UnmatchedAPIRouteReturnsJSON404(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		SolarController: config.SolarControllerConfiguration{
			HTTPPort: 8080,
			Epever:   epever.Configuration{Enabled: false},
		},
	}

	app, err := NewApplication(cfg, testutil.NewMockPublisher(), getTestVersionInfo())
	require.NoError(t, err)
	defer app.Close()

	for _, path := range []string{"/api/voltgo/metrics", "/api/epever/metrics", "/api/nonsense"} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", path, nil)
			app.Router().ServeHTTP(w, req)

			assert.Equal(t, http.StatusNotFound, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
			assert.NotContains(t, w.Body.String(), "<!doctype html>")
		})
	}
}

func TestApplication_AuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		SolarController: config.SolarControllerConfiguration{
			HTTPPort: 8080,
			Auth:     config.AuthConfiguration{Token: "secret-token"},
			Epever: epever.Configuration{
				Enabled: false,
			},
		},
	}

	app, err := NewApplication(cfg, testutil.NewMockPublisher(), getTestVersionInfo())
	require.NoError(t, err)
	defer app.Close()

	tests := []struct {
		name       string
		method     string
		path       string
		authHeader string
		wantCode   int
	}{
		{
			name:     "api request without token is rejected",
			method:   "GET",
			path:     "/api/info",
			wantCode: http.StatusUnauthorized,
		},
		{
			name:       "api request with wrong token is rejected",
			method:     "GET",
			path:       "/api/info",
			authHeader: "Bearer wrong-token",
			wantCode:   http.StatusUnauthorized,
		},
		{
			name:     "api write request without token is rejected",
			method:   "PATCH",
			path:     "/api/epever/battery-profile",
			wantCode: http.StatusUnauthorized,
		},
		{
			name:       "api request with valid token is allowed",
			method:     "GET",
			path:       "/api/info",
			authHeader: "Bearer secret-token",
			wantCode:   http.StatusOK,
		},
		{
			name:     "metrics without token is rejected",
			method:   "GET",
			path:     "/metrics",
			wantCode: http.StatusUnauthorized,
		},
		{
			name:       "metrics with valid token is allowed",
			method:     "GET",
			path:       "/metrics",
			authHeader: "Bearer secret-token",
			wantCode:   http.StatusOK,
		},
		{
			name:     "spa route stays public",
			method:   "GET",
			path:     "/",
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tt.method, tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			app.Router().ServeHTTP(w, req)
			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

func TestApplication_NoAuthTokenLeavesAPIOpen(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		SolarController: config.SolarControllerConfiguration{
			HTTPPort: 8080,
			Epever: epever.Configuration{
				Enabled: false,
			},
		},
	}

	app, err := NewApplication(cfg, testutil.NewMockPublisher(), getTestVersionInfo())
	require.NoError(t, err)
	defer app.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/info", nil)
	app.Router().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestApplication_GracefulShutdown(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		SolarController: config.SolarControllerConfiguration{
			HTTPPort:    0, // let the OS pick a free port
			BindAddress: "127.0.0.1",
			Epever: epever.Configuration{
				Enabled: false,
			},
		},
	}

	app, err := NewApplication(cfg, testutil.NewMockPublisher(), getTestVersionInfo())
	require.NoError(t, err)
	defer app.Close()

	runErr := make(chan error, 1)
	go func() { runErr <- app.Run() }()

	// Give the listener a moment to start, then shut down
	time.Sleep(100 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, app.Shutdown(ctx))

	select {
	case err := <-runErr:
		assert.NoError(t, err, "Run should return nil on clean shutdown")
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after Shutdown")
	}
}

func TestNewHTTPServer_SetsTimeouts(t *testing.T) {
	srv := newHTTPServer("127.0.0.1:0", http.NewServeMux())

	assert.Equal(t, "127.0.0.1:0", srv.Addr)
	assert.NotZero(t, srv.ReadHeaderTimeout)
	assert.NotZero(t, srv.ReadTimeout)
	assert.NotZero(t, srv.WriteTimeout)
	assert.NotZero(t, srv.IdleTimeout)
	assert.NotZero(t, srv.MaxHeaderBytes)
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

			mockPublisher := testutil.NewMockPublisher()

			app, err := NewApplication(cfg, mockPublisher, getTestVersionInfo())
			require.NoError(t, err)
			defer app.Close()

			// Test that epever endpoint is registered (or not)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/epever/metrics", nil)
			app.Router().ServeHTTP(w, req)

			if tt.expectEndpoint {
				// Endpoint should return JSON metrics
				assert.NotEqual(t, http.StatusNotFound, w.Code)
				assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
			} else {
				// Unmatched /api routes are answered with a JSON 404 rather than
				// the SPA fallback, so the frontend can tell a disabled
				// controller from one that is merely slow to report.
				assert.Equal(t, http.StatusNotFound, w.Code)
				assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
			}
		})
	}
}
