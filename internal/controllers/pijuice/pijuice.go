package pijuice

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"github.com/lumberbarons/solar-controller/internal/publisher"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	namespace = "pijuice"
)

type Configuration struct {
	I2cBus        string `yaml:"i2cBus"`
	I2cAddress    string `yaml:"i2cAddress"`
	PublishPeriod int    `yaml:"publishPeriod"`
}

type Controller struct {
	collector           *Collector
	configurer          *Configurer
	mqttPublisher       *publisher.MqttPublisher
	prometheusCollector *PrometheusCollector
	lastStatus          *ControllerStatus
}

func NewController(config Configuration, mqttPublisher *publisher.MqttPublisher) (*Controller, error) {
	if config.I2cAddress == "" {
		log.Info("pijuice disabled, no i2c address provided")
		return &Controller{}, nil
	}

	pijuiceCollector, err := NewCollector(config.I2cBus, config.I2cAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to start pijuice hat collector %w", err)
	}

	pijuiceConfigurer := NewConfigurer()

	log.Infof("connected to pijuice, i2c address %s", config.I2cAddress)

	controller := &Controller{
		collector:           pijuiceCollector,
		configurer:          pijuiceConfigurer,
		prometheusCollector: NewPrometheusCollector(),
		mqttPublisher:       mqttPublisher,
	}

	s := gocron.NewScheduler(time.UTC)

	_, err = s.Every(config.PublishPeriod).Seconds().WaitForSchedule().Do(controller.collectAndPublish)
	if err != nil {
		return nil, fmt.Errorf("failed to start pijuice hat publisher %w", err)
	}

	s.StartAsync()

	return controller, nil
}

func (e *Controller) collectAndPublish() {
	log.Info("collecting and publishing metrics for pijuice hat")

	status, err := e.collector.GetStatus()
	if err != nil {
		log.Errorf("failed to collect metrics from pijuice hat: %s", err)
		e.prometheusCollector.IncrementFailures()
		return
	}

	e.lastStatus = status
	e.prometheusCollector.SetMetrics(status)

	b, err := json.Marshal(status)
	if err != nil {
		log.Errorf("failed to marshall status for publishing for pijuice hat: %s", err)
		return
	}

	e.mqttPublisher.Publish(namespace, string(b))

	log.Info("collection done for pijuice hat")
}

func (e *Controller) MetricsGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, e.lastStatus)
	}
}

func (e *Controller) RegisterEndpoints(r *gin.Engine) {
	if !e.Enabled() {
		return
	}

	prefix := fmt.Sprintf("/api/%s", namespace)

	r.GET(prefix, func(c *gin.Context) {
		c.JSON(http.StatusOK, "{}")
	})

	r.GET(fmt.Sprintf("%s/metrics", prefix), e.MetricsGet())
}

func (e *Controller) Enabled() bool {
	return false
}

func (e *Controller) Close() {

}
