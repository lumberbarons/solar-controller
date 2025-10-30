package victron

import (
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
	namespace = "victron"
)

type Configuration struct {
	Enabled       bool   `yaml:"enabled"`
	MacAddress    string `yaml:"macAddress"`
	PublishPeriod int    `yaml:"publishPeriod"`
}

type Controller struct {
	collector           *Collector
	mqttPublisher       *publisher.MqttPublisher
	prometheusCollector *PrometheusCollector
	lastStatus          *ControllerStatus
}

func NewController(config Configuration, mqttPublisher *publisher.MqttPublisher) (*Controller, error) {
	if !config.Enabled {
		log.Info("victron disabled via configuration")
		return &Controller{}, nil
	}

	if config.MacAddress == "" {
		log.Warn("victron enabled but no mac address provided")
		return &Controller{}, nil
	}

	s := gocron.NewScheduler(time.UTC)

	victronCollector, err := NewCollector(config, s)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to victron: %w", err)
	}

	log.Infof("connected to victron %s", config.MacAddress)

	controller := &Controller{
		collector:           victronCollector,
		mqttPublisher:       mqttPublisher,
		prometheusCollector: NewPrometheusCollector(),
	}

	_, err = s.Every(config.PublishPeriod).Seconds().Do(controller.collectAndPublish)
	if err != nil {
		return nil, fmt.Errorf("failed to start victron publisher %w", err)
	}

	s.StartAsync()

	return controller, nil
}

func (e *Controller) collectAndPublish() {
	log.Info("collecting and publishing metrics for victron controller")

	status, err := e.collector.GetStatus()
	if err != nil {
		log.Errorf("failed to collect metrics from victron: %s", err)
		e.prometheusCollector.IncrementFailures()
		return
	}

	e.lastStatus = status
	e.prometheusCollector.SetMetrics(status)

	b, err := json.Marshal(status)
	if err != nil {
		log.Errorf("failed to marshall status for publishing for victron: %s", err)
		return
	}

	e.mqttPublisher.Publish(namespace, string(b))

	log.Info("collection done for victron controller")
}

func (e *Controller) RegisterEndpoints(r *gin.Engine) {
	if e.collector == nil {
		return
	}

	prefix := fmt.Sprintf("/api/%s", namespace)

	r.GET(prefix, func(c *gin.Context) {
		c.JSON(http.StatusOK, "{}")
	})

	r.GET(fmt.Sprintf("%s/metrics", prefix), func(c *gin.Context) {
		c.JSON(http.StatusOK, e.lastStatus)
	})
}

func (e *Controller) Enabled() bool {
	return e.collector != nil
}

func (e *Controller) Close() {

}
