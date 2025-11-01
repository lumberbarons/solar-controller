package epever

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"github.com/lumberbarons/solar-controller/internal/controllers"
	log "github.com/sirupsen/logrus"
)

const (
	namespace = "solar" // legacy
)

type Configuration struct {
	Enabled       bool   `yaml:"enabled"`
	SerialPort    string `yaml:"serialPort"`
	PublishPeriod int    `yaml:"publishPeriod"`
}

type Controller struct {
	client              controllers.ModbusClient
	collector           *Collector
	configurer          *Configurer
	mqttPublisher       controllers.MessagePublisher
	prometheusCollector controllers.MetricsCollector
	scheduler           *gocron.Scheduler
	lastStatus          *ControllerStatus
	lastStatusMutex     sync.RWMutex
	collectInProgress   bool
	collectMutex        sync.Mutex
}

// NewController creates a new Epever controller with dependency injection for testing.
// For production use, call NewControllerFromConfig instead.
func NewController(
	client controllers.ModbusClient,
	collector *Collector,
	configurer *Configurer,
	mqttPublisher controllers.MessagePublisher,
	prometheusCollector controllers.MetricsCollector,
	publishPeriod int,
) (*Controller, error) {
	if client == nil {
		return &Controller{}, nil
	}

	s := gocron.NewScheduler(time.UTC)

	controller := &Controller{
		client:              client,
		collector:           collector,
		configurer:          configurer,
		prometheusCollector: prometheusCollector,
		mqttPublisher:       mqttPublisher,
		scheduler:           s,
	}

	_, err := s.Every(publishPeriod).Seconds().Do(controller.collectAndPublish)
	if err != nil {
		return nil, fmt.Errorf("failed to start epever publisher %w", err)
	}

	s.StartAsync()

	// Run initial collection immediately
	go controller.collectAndPublish()

	return controller, nil
}

// NewControllerFromConfig creates a new Epever controller from configuration.
// This is the production entry point that creates all concrete dependencies.
func NewControllerFromConfig(config Configuration, mqttPublisher controllers.MessagePublisher) (*Controller, error) {
	if !config.Enabled {
		log.Info("epever disabled via configuration")
		return &Controller{}, nil
	}

	if config.SerialPort == "" {
		log.Warn("epever enabled but no serial port provided")
		return &Controller{}, nil
	}

	client, err := NewModbusClient(config.SerialPort)
	if err != nil {
		return nil, err
	}

	prometheusCollector := NewPrometheusCollector()
	epeverCollector := NewCollector(client)
	epeverConfigurer := NewConfigurer(client, prometheusCollector)

	log.Infof("connected to epever %s", config.SerialPort)

	return NewController(
		client,
		epeverCollector,
		epeverConfigurer,
		mqttPublisher,
		prometheusCollector,
		config.PublishPeriod,
	)
}

func (e *Controller) collectAndPublish() {
	// Check if a collection is already in progress
	e.collectMutex.Lock()
	if e.collectInProgress {
		log.Warn("collection already in progress for epever controller, skipping this collection cycle")
		e.collectMutex.Unlock()
		return
	}
	e.collectInProgress = true
	e.collectMutex.Unlock()

	// Ensure we clear the flag when done
	defer func() {
		e.collectMutex.Lock()
		e.collectInProgress = false
		e.collectMutex.Unlock()
	}()

	log.Debug("collecting and publishing metrics for epever controller")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	status, err := e.collector.GetStatus(ctx)
	if err != nil {
		log.Errorf("failed to collect metrics from epever: %s", err)
		e.prometheusCollector.IncrementFailures()
		return
	}

	e.lastStatusMutex.Lock()
	e.lastStatus = status
	e.lastStatusMutex.Unlock()

	e.prometheusCollector.SetMetrics(status)

	b, err := json.Marshal(status)
	if err != nil {
		log.Errorf("failed to marshall status for publishing for epever: %s", err)
		return
	}

	e.mqttPublisher.Publish(namespace, string(b))

	log.Debug("collection done for epever controller")
}

func (e *Controller) MetricsGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		e.lastStatusMutex.RLock()
		status := e.lastStatus
		e.lastStatusMutex.RUnlock()

		if status == nil {
			c.JSON(http.StatusNoContent, gin.H{})
			return
		}
		c.JSON(http.StatusOK, status)
	}
}

func (e *Controller) RegisterEndpoints(r *gin.Engine) {
	if e.client == nil {
		return
	}

	prefix := fmt.Sprintf("/api/%s", namespace)

	r.GET(prefix, func(c *gin.Context) {
		c.JSON(http.StatusOK, "{}")
	})

	r.GET(fmt.Sprintf("%s/metrics", prefix), e.MetricsGet())

	// New split configuration endpoints
	r.GET(fmt.Sprintf("%s/battery-profile", prefix), e.configurer.BatteryProfileGet())
	r.PATCH(fmt.Sprintf("%s/battery-profile", prefix), e.configurer.BatteryProfilePatch())
	r.GET(fmt.Sprintf("%s/charging-parameters", prefix), e.configurer.ChargingParametersGet())
	r.PATCH(fmt.Sprintf("%s/charging-parameters", prefix), e.configurer.ChargingParametersPatch())
	r.GET(fmt.Sprintf("%s/time", prefix), e.configurer.TimeGet())
	r.PATCH(fmt.Sprintf("%s/time", prefix), e.configurer.TimePatch())

	// Legacy endpoint (kept for backwards compatibility)
	r.GET(fmt.Sprintf("%s/config", prefix), e.configurer.ConfigGet())
	r.PATCH(fmt.Sprintf("%s/config", prefix), e.configurer.ConfigPatch())
}

func (e *Controller) Enabled() bool {
	return e.client != nil
}

func (e *Controller) Close() error {
	if e.scheduler != nil {
		e.scheduler.Stop()
		log.Debug("epever scheduler stopped")
	}
	if e.client != nil {
		e.client.Close()
	}
	return nil
}
