package remotewrite

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/klauspost/compress/snappy"
	"github.com/prometheus/prometheus/prompb"
	log "github.com/sirupsen/logrus"
)

// Publisher publishes metrics to a Prometheus remote_write endpoint.
type Publisher struct {
	config      *Configuration
	httpClient  *http.Client
	topicPrefix string
	deviceID    string

	// Batching support
	mu              sync.Mutex
	batchBuffer     []metricData
	lastPublishTime time.Time
	batchTimeout    time.Duration
}

// metricData holds parsed metric information from a publish call.
type metricData struct {
	metricName string
	labels     map[string]string
	value      float64
	timestamp  int64
}

// MetricPayload represents the JSON payload structure from controllers.
type MetricPayload struct {
	Value     interface{} `json:"value"`
	Unit      string      `json:"unit"`
	Timestamp int64       `json:"timestamp"`
}

// NewPublisher creates a new remote_write publisher.
func NewPublisher(config *Configuration, topicPrefix, deviceID string) (*Publisher, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid remote_write configuration: %w", err)
	}

	if !config.Enabled {
		log.Warn("Remote write publisher created but not enabled")
		return &Publisher{}, nil
	}

	httpClient := &http.Client{
		Timeout: config.GetTimeout(),
	}

	p := &Publisher{
		config:          config,
		httpClient:      httpClient,
		topicPrefix:     topicPrefix,
		deviceID:        deviceID,
		batchBuffer:     make([]metricData, 0, 20), // Pre-allocate for ~12 metrics
		batchTimeout:    5 * time.Second,           // Max time to hold metrics before sending
		lastPublishTime: time.Now(),
	}

	log.WithFields(log.Fields{
		"url":         config.URL,
		"timeout":     config.GetTimeout(),
		"topicPrefix": topicPrefix,
		"deviceID":    deviceID,
	}).Info("Remote write publisher initialized")

	return p, nil
}

// Publish implements the MessagePublisher interface.
// It buffers metrics and sends them as a batch when appropriate.
func (p *Publisher) Publish(topicSuffix, payload string) {
	if p.httpClient == nil {
		return // Publisher not enabled
	}

	// Parse the metric from the topic and payload
	metric, err := p.parseMetric(topicSuffix, payload)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"topicSuffix": topicSuffix,
			"payload":     payload,
		}).Error("Failed to parse metric for remote write")
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Add to batch buffer
	p.batchBuffer = append(p.batchBuffer, metric)

	// Determine if we should flush the batch
	// Strategy: Flush when we detect all metrics from a collection cycle
	// Heuristic: If we have 12+ metrics (Epever has 12) or timeout elapsed
	shouldFlush := len(p.batchBuffer) >= 12 ||
		time.Since(p.lastPublishTime) > p.batchTimeout

	if shouldFlush {
		p.flushBatch()
	}
}

// parseMetric converts a topic suffix and JSON payload into metric data.
// Topic format: {deviceId}/{controller}/{metric-name}
// Example: controller-123/epever/battery-voltage
func (p *Publisher) parseMetric(topicSuffix, payload string) (metricData, error) {
	// Parse topic suffix
	parts := strings.Split(topicSuffix, "/")
	if len(parts) != 3 {
		return metricData{}, fmt.Errorf("invalid topic format, expected 3 parts: %s", topicSuffix)
	}

	deviceID := parts[0]
	controller := parts[1]
	metricNameKebab := parts[2]

	// Convert kebab-case to snake_case for Prometheus naming
	metricNameSnake := strings.ReplaceAll(metricNameKebab, "-", "_")

	// Construct Prometheus metric name: {controller}_{metric_name}
	fullMetricName := fmt.Sprintf("%s_%s", controller, metricNameSnake)

	// Parse JSON payload
	var metricPayload MetricPayload
	if err := json.Unmarshal([]byte(payload), &metricPayload); err != nil {
		return metricData{}, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Convert value to float64
	value, err := toFloat64(metricPayload.Value)
	if err != nil {
		return metricData{}, fmt.Errorf("failed to convert value to float64: %w", err)
	}

	// Build labels
	labels := map[string]string{
		"device_id":  deviceID,
		"controller": controller,
	}

	// Add unit as a label if present
	if metricPayload.Unit != "" {
		labels["unit"] = metricPayload.Unit
	}

	return metricData{
		metricName: fullMetricName,
		labels:     labels,
		value:      value,
		timestamp:  metricPayload.Timestamp,
	}, nil
}

// flushBatch sends the buffered metrics to the remote_write endpoint.
// Must be called with p.mu locked.
func (p *Publisher) flushBatch() {
	if len(p.batchBuffer) == 0 {
		return
	}

	// Convert metrics to Prometheus TimeSeries
	timeSeries := p.metricsToTimeSeries(p.batchBuffer)

	// Create WriteRequest
	writeRequest := &prompb.WriteRequest{
		Timeseries: timeSeries,
	}

	// Marshal to protobuf
	data, err := writeRequest.Marshal()
	if err != nil {
		log.WithError(err).Error("Failed to marshal WriteRequest to protobuf")
		p.batchBuffer = p.batchBuffer[:0] // Clear buffer even on error
		return
	}

	// Compress with Snappy
	compressed := snappy.Encode(nil, data)

	// Send HTTP request
	if err := p.sendRequest(compressed); err != nil {
		log.WithError(err).WithField("metricsCount", len(p.batchBuffer)).Error("Failed to send remote write request")
	} else {
		log.WithField("metricsCount", len(p.batchBuffer)).Debug("Successfully sent remote write batch")
	}

	// Clear buffer and update timestamp
	p.batchBuffer = p.batchBuffer[:0]
	p.lastPublishTime = time.Now()
}

// metricsToTimeSeries converts internal metric data to Prometheus TimeSeries format.
func (p *Publisher) metricsToTimeSeries(metrics []metricData) []prompb.TimeSeries {
	// Group metrics by metric name and labels (same series)
	seriesMap := make(map[string]*prompb.TimeSeries)

	for _, metric := range metrics {
		// Create a key for this series (metric name + sorted labels)
		seriesKey := p.seriesKey(metric.metricName, metric.labels)

		ts, exists := seriesMap[seriesKey]
		if !exists {
			// Create new TimeSeries
			labels := make([]prompb.Label, 0, len(metric.labels)+1)

			// Add __name__ label for metric name
			labels = append(labels, prompb.Label{
				Name:  "__name__",
				Value: metric.metricName,
			})

			// Add other labels (sorted for consistency)
			for k, v := range metric.labels {
				labels = append(labels, prompb.Label{
					Name:  k,
					Value: v,
				})
			}

			ts = &prompb.TimeSeries{
				Labels:  labels,
				Samples: make([]prompb.Sample, 0, 1),
			}
			seriesMap[seriesKey] = ts
		}

		// Add sample to the series
		// Note: Prometheus remote_write expects timestamps in milliseconds
		ts.Samples = append(ts.Samples, prompb.Sample{
			Value:     metric.value,
			Timestamp: metric.timestamp * 1000, // Convert seconds to milliseconds
		})
	}

	// Convert map to slice
	result := make([]prompb.TimeSeries, 0, len(seriesMap))
	for _, ts := range seriesMap {
		result = append(result, *ts)
	}

	return result
}

// seriesKey creates a unique key for a time series.
func (p *Publisher) seriesKey(metricName string, labels map[string]string) string {
	// Simple key: metricName + comma-separated labels
	parts := []string{metricName}
	for k, v := range labels {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, ",")
}

// sendRequest sends the compressed protobuf data to the remote_write endpoint.
func (p *Publisher) sendRequest(compressedData []byte) error {
	req, err := http.NewRequest("POST", p.config.URL, bytes.NewReader(compressedData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set required headers
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")
	req.Header.Set("User-Agent", "solar-controller/1.0")

	// Add authentication
	if p.config.BasicAuth != nil {
		req.SetBasicAuth(p.config.BasicAuth.Username, p.config.BasicAuth.Password)
	} else if p.config.BearerToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.BearerToken))
	}

	// Add custom headers
	for k, v := range p.config.Headers {
		req.Header.Set(k, v)
	}

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for logging
	body, _ := io.ReadAll(resp.Body) // nolint:errcheck // Body is only used for error logging

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("remote write failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Close implements the MessagePublisher interface.
// It flushes any remaining buffered metrics.
func (p *Publisher) Close() {
	if p.httpClient == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Flush any remaining metrics
	if len(p.batchBuffer) > 0 {
		log.WithField("metricsCount", len(p.batchBuffer)).Info("Flushing remaining metrics on close")
		p.flushBatch()
	}

	log.Info("Remote write publisher closed")
}

// toFloat64 converts various numeric types to float64.
func toFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("unsupported value type: %T", value)
	}
}
