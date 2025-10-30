package epever

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"github.com/lumberbarons/solar-controller/internal/publisher"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
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
	client              *ModbusClient
	collector           *Collector
	configurer          *Configurer
	mqttPublisher       *publisher.MqttPublisher
	prometheusCollector *PrometheusCollector
	lastStatus          *ControllerStatus
}

func NewController(config Configuration, mqttPublisher *publisher.MqttPublisher) (*Controller, error) {
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

	epeverCollector := NewCollector(client)
	epeverConfigurer := NewConfigurer(client)

	log.Infof("connected to epever %s", config.SerialPort)

	controller := &Controller{
		client:              client,
		collector:           epeverCollector,
		configurer:          epeverConfigurer,
		prometheusCollector: NewPrometheusCollector(),
		mqttPublisher:       mqttPublisher,
	}

	s := gocron.NewScheduler(time.UTC)

	_, err = s.Every(config.PublishPeriod).Seconds().WaitForSchedule().Do(controller.collectAndPublish)
	if err != nil {
		return nil, fmt.Errorf("failed to start epever publisher %w", err)
	}

	s.StartAsync()

	return controller, nil
}

func (e *Controller) collectAndPublish() {
	log.Trace("collecting and publishing metrics for epever controller")

	ctx := context.Background()
	status, err := e.collector.GetStatus(ctx)
	if err != nil {
		log.Errorf("failed to collect metrics from epever: %s", err)
		e.prometheusCollector.IncrementFailures()
		return
	}

	e.lastStatus = status
	e.prometheusCollector.SetMetrics(status)

	b, err := json.Marshal(status)
	if err != nil {
		log.Errorf("failed to marshall status for publishing for epever: %s", err)
		return
	}

	e.mqttPublisher.Publish(namespace, string(b))

	log.Trace("collection done for epever controller")
}

func (e *Controller) MetricsGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		if e.lastStatus == nil {
			c.JSON(http.StatusNoContent, gin.H{})
		} else {
			c.JSON(http.StatusOK, e.lastStatus)
		}
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

	r.POST(fmt.Sprintf("%s/query", prefix), e.configurer.QueryPost())
}

func (e *Controller) Enabled() bool {
	return e.client != nil
}

func (e *Controller) Close() {
	if e.client != nil {
		e.client.Close()
	}
}
