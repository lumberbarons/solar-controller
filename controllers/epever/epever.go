package epever

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"github.com/goburrow/modbus"
	"github.com/lumberbarons/solar-controller/publisher"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

const (
	namespace = "epever"
)

type Configuration struct {
	SerialPort    string `yaml:"serialPort"`
	PublishPeriod int    `yaml:"publishPeriod"`
}

type Controller struct {
	handler *modbus.RTUClientHandler
	collector *Collector
	configurer *Configurer
	mqttPublisher *publisher.MqttPublisher
	prometheusCollector *PrometheusCollector

	lastStatus *ControllerStatus
}

func NewController(config Configuration, mqttPublisher *publisher.MqttPublisher) (*Controller, error) {
	if config.SerialPort == "" {
		log.Info("epever disabled, no serial port provided")
		return &Controller{}, nil
	}

	handler := modbus.NewRTUClientHandler(config.SerialPort)

	handler.BaudRate = 115200
	handler.DataBits = 8
	handler.Parity = "N"
	handler.StopBits = 1
	handler.SlaveId = 1
	handler.Timeout = 2 * time.Second

	err := handler.Connect()

	if err != nil {
		return nil, fmt.Errorf("failed to connect to epever: %w", err)
	}

	client := modbus.NewClient(handler)

	epeverCollector := NewCollector(client)
	epeverConfigurer := NewConfigurer(client)
	prometheusCollector := NewPrometheusCollector()

	log.Infof("connected to epever %s", config.SerialPort)

	controller := &Controller{
		handler: handler,
		collector: epeverCollector,
		configurer: epeverConfigurer,
		prometheusCollector: prometheusCollector,
		mqttPublisher: mqttPublisher,
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

	status, err := e.collector.GetStatus()
	if err != nil {
		log.Errorf("failed to collect metrics from epever: %s", err)
		e.prometheusCollector.IncrementFailures()
		return
	}

	e.lastStatus = status
	e.prometheusCollector.SetMetrics(status)

	b, err := json.Marshal(status)
	if err != nil {
		log.Errorf("failed to collect marshall status for publishing for epever: %s", err)
		return
	}

	e.mqttPublisher.Publish(namespace, string(b))
}

func (e *Controller) MetricsGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, e.lastStatus)
	}
}

func (e *Controller) RegisterEndpoints(r *gin.Engine) {
	if e.handler == nil {
		return
	}

	r.GET("/api/epever", func(c *gin.Context) {
		c.JSON(http.StatusOK, "{}")
	})

	r.GET("/api/epever/metrics", e.MetricsGet())
	r.GET("/api/epever/config", e.configurer.ConfigGet())
	r.PATCH("/api/epever/config", e.configurer.ConfigPatch())
	r.POST("/api/epever/query", e.configurer.QueryPost())
}

func (e *Controller) Enabled() bool {
	return e.configurer != nil
}

func (e *Controller) Close() {
	if e.handler != nil {
		e.handler.Close()
	}
}
