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
	if config.SerialPort == "" {
		log.Info("epever disabled, no serial port provided")
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
	log.Info("collecting and publishing metrics for epever controller")

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

	log.Info("collection done for epever controller")
}

func (e *Controller) MetricsGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, e.lastStatus)
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
