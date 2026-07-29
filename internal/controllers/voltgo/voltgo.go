package voltgo

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"github.com/lumberbarons/solar-controller/internal/publish"
	log "github.com/sirupsen/logrus"
)

const (
	namespace = "voltgo"

	defaultConnectTimeout = 30 * time.Second

	// collectTimeout bounds a full collection cycle: a BLE connect
	// (default 30s) plus the status reads.
	collectTimeout = 60 * time.Second
)

type Configuration struct {
	Enabled       bool   `yaml:"enabled"`
	Address       string `yaml:"address"`
	PublishPeriod int    `yaml:"publishPeriod"`

	// ConnectTimeout is the maximum time to wait for a BLE connection,
	// as a duration string (default: 30s)
	ConnectTimeout string `yaml:"connectTimeout"`
}

// Validate checks the configuration for errors. Only called when enabled.
func (c *Configuration) Validate() error {
	if c.ConnectTimeout != "" {
		if _, err := time.ParseDuration(c.ConnectTimeout); err != nil {
			return err
		}
	}
	return nil
}

// GetConnectTimeout returns the configured connect timeout or the default (30s).
func (c *Configuration) GetConnectTimeout() time.Duration {
	if c.ConnectTimeout == "" {
		return defaultConnectTimeout
	}
	timeout, err := time.ParseDuration(c.ConnectTimeout)
	if err != nil {
		return defaultConnectTimeout
	}
	return timeout
}

type Controller struct {
	collector           *Collector
	publisher           publish.MessagePublisher
	prometheusCollector MetricsCollector
	scheduler           *gocron.Scheduler
	deviceID            string
	lastStatus          *BatteryStatus
	lastInfo            *BatteryInfo
	lastStatusMutex     sync.RWMutex
	collectInProgress   bool
	collectMutex        sync.Mutex
}

// NewController creates a new voltgo controller with dependency injection for testing.
// For production use, call NewControllerFromConfig instead.
func NewController(
	collector *Collector,
	publisher publish.MessagePublisher,
	prometheusCollector MetricsCollector,
	deviceID string,
	publishPeriod int,
) (*Controller, error) {
	if collector == nil {
		return &Controller{}, nil
	}

	// Default device ID if not provided
	if deviceID == "" {
		deviceID = "controller-1"
	}

	s := gocron.NewScheduler(time.UTC)

	controller := &Controller{
		collector:           collector,
		publisher:           publisher,
		prometheusCollector: prometheusCollector,
		deviceID:            deviceID,
		scheduler:           s,
	}

	_, err := s.Every(publishPeriod).Seconds().Do(controller.collectAndPublish)
	if err != nil {
		return nil, fmt.Errorf("failed to start voltgo publisher %w", err)
	}

	s.StartAsync()

	// Run initial collection immediately
	go controller.collectAndPublish()

	return controller, nil
}

// newControllerForTest creates a Controller without starting the scheduler or background goroutine.
// This allows tests to call collectAndPublish synchronously without racing.
func newControllerForTest(
	collector *Collector,
	publisher publish.MessagePublisher,
	prometheusCollector MetricsCollector,
	deviceID string,
) *Controller {
	if deviceID == "" {
		deviceID = "controller-1"
	}
	return &Controller{
		collector:           collector,
		publisher:           publisher,
		prometheusCollector: prometheusCollector,
		deviceID:            deviceID,
	}
}

// NewControllerFromConfig creates a new voltgo controller from configuration.
// This is the production entry point that creates all concrete dependencies.
func NewControllerFromConfig(config Configuration, publisher publish.MessagePublisher, deviceID string) (*Controller, error) {
	if !config.Enabled {
		log.Info("voltgo disabled via configuration")
		return &Controller{}, nil
	}

	if config.Address == "" {
		log.Warn("voltgo enabled but no battery address provided")
		return &Controller{}, nil
	}

	connector, err := NewBLEConnector()
	if err != nil {
		return nil, err
	}

	collector := NewCollector(connector, config.Address, config.GetConnectTimeout())
	prometheusCollector := NewPrometheusCollector()

	log.Infof("voltgo battery configured at %s", config.Address)

	return NewController(
		collector,
		publisher,
		prometheusCollector,
		deviceID,
		config.PublishPeriod,
	)
}

func (v *Controller) collectAndPublish() {
	// Check if a collection is already in progress
	v.collectMutex.Lock()
	if v.collectInProgress {
		log.Warn("collection already in progress for voltgo controller, skipping this collection cycle")
		v.collectMutex.Unlock()
		return
	}
	v.collectInProgress = true
	v.collectMutex.Unlock()

	// Ensure we clear the flag when done
	defer func() {
		v.collectMutex.Lock()
		v.collectInProgress = false
		v.collectMutex.Unlock()
	}()

	log.Debug("collecting and publishing metrics for voltgo controller")

	ctx, cancel := context.WithTimeout(context.Background(), collectTimeout)
	defer cancel()

	status, err := v.collector.GetStatus(ctx)
	if err != nil {
		log.Errorf("failed to collect metrics from voltgo battery: %s", err)
		v.prometheusCollector.IncrementFailures()

		// Publish failure metric to message broker
		failureMetric := CreateCollectionFailureMetric()
		payload, err := failureMetric.ToJSON()
		if err != nil {
			log.Errorf("failed to marshal failure metric: %s", err)
			return
		}

		topicSuffix := fmt.Sprintf("%s/%s/%s", v.deviceID, namespace, failureMetric.Name)
		v.publisher.Publish(topicSuffix, payload)
		log.Debugf("published failure metric to %s", topicSuffix)

		return
	}

	v.lastStatusMutex.Lock()
	v.lastStatus = status
	v.lastStatusMutex.Unlock()

	v.prometheusCollector.SetMetrics(status)

	// Fetch static battery info once, now that a connection is up
	v.fetchInfoOnce(ctx)

	// Convert status to individual metrics
	metrics := ConvertStatusToMetrics(status)

	// Publish each metric individually
	for _, metric := range metrics {
		payload, err := metric.ToJSON()
		if err != nil {
			log.Errorf("failed to marshal metric %s for publishing: %s", metric.Name, err)
			continue
		}

		// Topic format: {deviceId}/voltgo/{metric-name}
		topicSuffix := fmt.Sprintf("%s/%s/%s", v.deviceID, namespace, metric.Name)
		v.publisher.Publish(topicSuffix, payload)

		log.Debugf("published metric %s to %s", metric.Name, topicSuffix)
	}

	log.Debug("collection done for voltgo controller")
}

// fetchInfoOnce caches static battery info on the first successful collection
// cycle. Failures are logged but never fail the cycle - the next cycle retries.
func (v *Controller) fetchInfoOnce(ctx context.Context) {
	v.lastStatusMutex.RLock()
	cached := v.lastInfo != nil
	v.lastStatusMutex.RUnlock()
	if cached {
		return
	}

	info, err := v.collector.GetInfo(ctx)
	if err != nil {
		log.Warnf("failed to read voltgo battery info: %s", err)
		return
	}

	v.lastStatusMutex.Lock()
	v.lastInfo = info
	v.lastStatusMutex.Unlock()
	log.Debugf("cached voltgo battery info: %s %.1fV %.0fAh", info.Chemistry, info.NominalVoltage, info.CapacityAh)
}

func (v *Controller) MetricsGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		v.lastStatusMutex.RLock()
		status := v.lastStatus
		v.lastStatusMutex.RUnlock()

		if status == nil {
			c.JSON(http.StatusNoContent, gin.H{})
			return
		}
		c.JSON(http.StatusOK, status)
	}
}

func (v *Controller) InfoGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		v.lastStatusMutex.RLock()
		info := v.lastInfo
		v.lastStatusMutex.RUnlock()

		if info == nil {
			c.JSON(http.StatusNoContent, gin.H{})
			return
		}
		c.JSON(http.StatusOK, info)
	}
}

func (v *Controller) RegisterEndpoints(r *gin.Engine) {
	if v.collector == nil {
		return
	}

	prefix := fmt.Sprintf("/api/%s", namespace)

	r.GET(fmt.Sprintf("%s/metrics", prefix), v.MetricsGet())
	r.GET(fmt.Sprintf("%s/info", prefix), v.InfoGet())
}

func (v *Controller) Enabled() bool {
	return v.collector != nil
}

func (v *Controller) Close() error {
	if v.scheduler != nil {
		v.scheduler.Stop()
		log.Debug("voltgo scheduler stopped")
	}
	if v.collector != nil {
		return v.collector.Close()
	}
	return nil
}
