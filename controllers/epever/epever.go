package epever

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/goburrow/modbus"
	"github.com/lumberbarons/solar-controller/common"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type Configuration struct {
	SerialPort  string `yaml:"serialPort"`
	CacheExpiry int64  `yaml:"cacheExpiry"`
}

type Controller struct {
	handler *modbus.RTUClientHandler
	Collector *Collector
	Configurer *Configurer
	PrometheusCollector *PrometheusCollector
}

func NewController(config Configuration) (*Controller, error) {
	if config.SerialPort == "" {
		log.Info("epever disabled, no serial port provided")
		return &Controller{},nil
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

	epeverCollector := NewCollector(client, config.CacheExpiry)
	epeverConfigurer := NewConfigurer(client)
	prometheusCollector := NewPrometheusCollector(epeverCollector)

	log.Infof("connected to epever %s", config.SerialPort)

	return &Controller{
		handler: handler,
		Collector: epeverCollector,
		Configurer: epeverConfigurer,
		PrometheusCollector: prometheusCollector,
	},nil
}

func (e *Controller) RegisterEndpoints(r *gin.Engine) {
	if e.handler == nil {
		return
	}

	r.GET("/api/epever", func(c *gin.Context) {
		c.JSON(http.StatusOK, "{}")
	})

	r.GET("/api/epever/metrics", e.Collector.MetricsGet())
	r.GET("/api/epever/config", e.Configurer.ConfigGet())
	r.PATCH("/api/epever/config", e.Configurer.ConfigPatch())
	r.POST("/api/epever/query", e.Configurer.QueryPost())
}

func (e *Controller) GetSolarCollector() common.SolarCollector {
	return e.Collector
}

func (e *Controller) GetPrometheusCollector() prometheus.Collector {
	return e.PrometheusCollector
}

func (e *Controller) Close() {
	if e.handler != nil {
		e.handler.Close()
	}
}
