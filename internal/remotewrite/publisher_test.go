package remotewrite

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/klauspost/compress/snappy"
	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPublisher(t *testing.T) {
	tests := []struct {
		name        string
		config      *Configuration
		topicPrefix string
		deviceID    string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid configuration with basic auth",
			config: &Configuration{
				Enabled: true,
				URL:     "http://localhost:9090/api/v1/write",
				BasicAuth: &BasicAuthConfig{
					Username: "user",
					Password: "pass",
				},
			},
			topicPrefix: "solar",
			deviceID:    "controller-1",
			wantErr:     false,
		},
		{
			name: "valid configuration with bearer token",
			config: &Configuration{
				Enabled:     true,
				URL:         "https://prometheus.example.com/api/v1/write",
				BearerToken: "token123",
			},
			topicPrefix: "solar",
			deviceID:    "controller-1",
			wantErr:     false,
		},
		{
			name: "disabled publisher",
			config: &Configuration{
				Enabled: false,
			},
			topicPrefix: "solar",
			deviceID:    "controller-1",
			wantErr:     false,
		},
		{
			name: "missing URL",
			config: &Configuration{
				Enabled: true,
			},
			topicPrefix: "solar",
			deviceID:    "controller-1",
			wantErr:     true,
			errContains: "url is required",
		},
		{
			name: "invalid URL scheme",
			config: &Configuration{
				Enabled: true,
				URL:     "ftp://localhost/api/v1/write",
			},
			topicPrefix: "solar",
			deviceID:    "controller-1",
			wantErr:     true,
			errContains: "must use http or https",
		},
		{
			name: "both basicAuth and bearerToken",
			config: &Configuration{
				Enabled:     true,
				URL:         "http://localhost:9090/api/v1/write",
				BearerToken: "token123",
				BasicAuth: &BasicAuthConfig{
					Username: "user",
					Password: "pass",
				},
			},
			topicPrefix: "solar",
			deviceID:    "controller-1",
			wantErr:     true,
			errContains: "mutually exclusive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publisher, err := NewPublisher(tt.config, tt.topicPrefix, tt.deviceID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, publisher)
			}
		})
	}
}

func TestParseMetric(t *testing.T) {
	publisher := &Publisher{
		topicPrefix: "solar",
		deviceID:    "controller-1",
	}

	tests := []struct {
		name        string
		topicSuffix string
		payload     string
		wantMetric  metricData
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid metric with float value",
			topicSuffix: "controller-123/epever/battery-voltage",
			payload:     `{"value": 12.5, "unit": "volts", "timestamp": 1699000000}`,
			wantMetric: metricData{
				metricName: "epever_battery_voltage",
				labels: map[string]string{
					"device_id":  "controller-123",
					"controller": "epever",
					"unit":       "volts",
				},
				value:     12.5,
				timestamp: 1699000000,
			},
			wantErr: false,
		},
		{
			name:        "valid metric with int value",
			topicSuffix: "controller-123/epever/battery-soc",
			payload:     `{"value": 85, "unit": "percent", "timestamp": 1699000001}`,
			wantMetric: metricData{
				metricName: "epever_battery_soc",
				labels: map[string]string{
					"device_id":  "controller-123",
					"controller": "epever",
					"unit":       "percent",
				},
				value:     85.0,
				timestamp: 1699000001,
			},
			wantErr: false,
		},
		{
			name:        "valid metric without unit",
			topicSuffix: "controller-123/epever/charging-status",
			payload:     `{"value": 1, "unit": "", "timestamp": 1699000002}`,
			wantMetric: metricData{
				metricName: "epever_charging_status",
				labels: map[string]string{
					"device_id":  "controller-123",
					"controller": "epever",
				},
				value:     1.0,
				timestamp: 1699000002,
			},
			wantErr: false,
		},
		{
			name:        "invalid topic format - too few parts",
			topicSuffix: "controller-123/epever",
			payload:     `{"value": 12.5, "unit": "volts", "timestamp": 1699000000}`,
			wantErr:     true,
			errContains: "invalid topic format",
		},
		{
			name:        "invalid JSON payload",
			topicSuffix: "controller-123/epever/battery-voltage",
			payload:     `{invalid json}`,
			wantErr:     true,
			errContains: "failed to unmarshal",
		},
		{
			name:        "invalid value type",
			topicSuffix: "controller-123/epever/battery-voltage",
			payload:     `{"value": "not a number", "unit": "volts", "timestamp": 1699000000}`,
			wantErr:     true,
			errContains: "failed to convert",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metric, err := publisher.parseMetric(tt.topicSuffix, tt.payload)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantMetric.metricName, metric.metricName)
				assert.Equal(t, tt.wantMetric.value, metric.value)
				assert.Equal(t, tt.wantMetric.timestamp, metric.timestamp)
				assert.Equal(t, tt.wantMetric.labels, metric.labels)
			}
		})
	}
}

func TestPublishWithMockServer(t *testing.T) {
	// Track requests received by the mock server
	var receivedRequests []receivedRequest
	var mu sync.Mutex

	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		// Read and decompress body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Decompress with snappy
		decompressed, err := snappy.Decode(nil, body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Failed to decompress: %v", err)
			return
		}

		// Unmarshal protobuf
		var writeRequest prompb.WriteRequest
		if err := writeRequest.Unmarshal(decompressed); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Failed to unmarshal: %v", err)
			return
		}

		// Record the request
		receivedRequests = append(receivedRequests, receivedRequest{
			headers:      r.Header.Clone(),
			writeRequest: writeRequest,
		})

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create publisher with mock server URL
	config := &Configuration{
		Enabled: true,
		URL:     server.URL,
		BasicAuth: &BasicAuthConfig{
			Username: "testuser",
			Password: "testpass",
		},
		Headers: map[string]string{
			"X-Scope-OrgID": "tenant1",
		},
	}

	publisher, err := NewPublisher(config, "solar", "controller-123")
	require.NoError(t, err)
	defer publisher.Close()

	// Publish a batch of metrics (simulating Epever collection)
	metrics := []struct {
		topicSuffix string
		payload     string
	}{
		{"controller-123/epever/array-voltage", `{"value": 18.5, "unit": "volts", "timestamp": 1699000000}`},
		{"controller-123/epever/array-current", `{"value": 5.2, "unit": "amperes", "timestamp": 1699000000}`},
		{"controller-123/epever/battery-voltage", `{"value": 12.4, "unit": "volts", "timestamp": 1699000000}`},
		{"controller-123/epever/battery-soc", `{"value": 85, "unit": "percent", "timestamp": 1699000000}`},
		{"controller-123/epever/battery-temp", `{"value": 25.3, "unit": "celsius", "timestamp": 1699000000}`},
		{"controller-123/epever/device-temp", `{"value": 28.7, "unit": "celsius", "timestamp": 1699000000}`},
		{"controller-123/epever/charging-power", `{"value": 96.2, "unit": "watts", "timestamp": 1699000000}`},
		{"controller-123/epever/charging-current", `{"value": 7.8, "unit": "amperes", "timestamp": 1699000000}`},
		{"controller-123/epever/array-power", `{"value": 96.2, "unit": "watts", "timestamp": 1699000000}`},
		{"controller-123/epever/energy-generated-daily", `{"value": 2.5, "unit": "kilowatt-hours", "timestamp": 1699000000}`},
		{"controller-123/epever/charging-status", `{"value": 1, "unit": "code", "timestamp": 1699000000}`},
		{"controller-123/epever/collection-time", `{"value": 0.352, "unit": "seconds", "timestamp": 1699000000}`},
	}

	for _, m := range metrics {
		publisher.Publish(m.topicSuffix, m.payload)
	}

	// Wait a bit for async processing
	time.Sleep(100 * time.Millisecond)

	// Verify requests were received
	mu.Lock()
	defer mu.Unlock()

	assert.Equal(t, 1, len(receivedRequests), "Expected 1 batch request")

	if len(receivedRequests) > 0 {
		req := receivedRequests[0]

		// Verify headers
		assert.Equal(t, "application/x-protobuf", req.headers.Get("Content-Type"))
		assert.Equal(t, "snappy", req.headers.Get("Content-Encoding"))
		assert.Equal(t, "0.1.0", req.headers.Get("X-Prometheus-Remote-Write-Version"))
		assert.Equal(t, "tenant1", req.headers.Get("X-Scope-OrgID"))

		// Verify basic auth header is present
		authHeader := req.headers.Get("Authorization")
		assert.NotEmpty(t, authHeader, "Authorization header should be present")

		// Verify timeseries count
		assert.Equal(t, 12, len(req.writeRequest.Timeseries), "Expected 12 time series")

		// Verify one of the metrics
		foundBatteryVoltage := false
		for _, ts := range req.writeRequest.Timeseries {
			// Find __name__ label
			var metricName string
			labels := make(map[string]string)
			for _, label := range ts.Labels {
				if label.Name == "__name__" {
					metricName = label.Value
				} else {
					labels[label.Name] = label.Value
				}
			}

			if metricName == "epever_battery_voltage" {
				foundBatteryVoltage = true
				assert.Equal(t, "controller-123", labels["device_id"])
				assert.Equal(t, "epever", labels["controller"])
				assert.Equal(t, "volts", labels["unit"])
				assert.Equal(t, 1, len(ts.Samples))
				assert.Equal(t, 12.4, ts.Samples[0].Value)
				// Timestamp should be in milliseconds (1699000000 seconds * 1000)
				assert.Equal(t, int64(1699000000000), ts.Samples[0].Timestamp)
			}
		}
		assert.True(t, foundBatteryVoltage, "Expected to find battery voltage metric")
	}
}

func TestPublishWithBearerToken(t *testing.T) {
	receivedAuth := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { // nolint:revive // r is used
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &Configuration{
		Enabled:     true,
		URL:         server.URL,
		BearerToken: "mytoken123",
	}

	publisher, err := NewPublisher(config, "solar", "controller-1")
	require.NoError(t, err)
	defer publisher.Close()

	// Publish enough metrics to trigger batch
	for i := 0; i < 12; i++ {
		publisher.Publish(
			fmt.Sprintf("controller-1/epever/metric-%d", i),
			fmt.Sprintf(`{"value": %d, "unit": "test", "timestamp": 1699000000}`, i),
		)
	}

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, "Bearer mytoken123", receivedAuth)
}

func TestPublishError(t *testing.T) {
	// Server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal server error")) // nolint:errcheck // Test code
	}))
	defer server.Close()

	config := &Configuration{
		Enabled: true,
		URL:     server.URL,
	}

	publisher, err := NewPublisher(config, "solar", "controller-1")
	require.NoError(t, err)
	defer publisher.Close()

	// This should not panic, just log the error
	for i := 0; i < 12; i++ {
		publisher.Publish(
			fmt.Sprintf("controller-1/epever/metric-%d", i),
			fmt.Sprintf(`{"value": %d, "unit": "test", "timestamp": 1699000000}`, i),
		)
	}

	time.Sleep(100 * time.Millisecond)
	// Test passes if no panic
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		want    float64
		wantErr bool
	}{
		{"float64", float64(12.5), 12.5, false},
		{"float32", float32(12.5), 12.5, false},
		{"int", int(10), 10.0, false},
		{"int32", int32(10), 10.0, false},
		{"int64", int64(10), 10.0, false},
		{"uint", uint(10), 10.0, false},
		{"uint32", uint32(10), 10.0, false},
		{"uint64", uint64(10), 10.0, false},
		{"string", "not a number", 0, true},
		{"bool", true, 0, true},
		{"nil", nil, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toFloat64(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// Helper types
type receivedRequest struct {
	headers      http.Header
	writeRequest prompb.WriteRequest
}
