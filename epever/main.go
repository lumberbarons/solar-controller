package epever

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/goburrow/modbus"
	"github.com/lumberbarons/solar-controller/epever/collector"
	"github.com/lumberbarons/solar-controller/epever/configurer"
	log "github.com/sirupsen/logrus"
	"time"
)

type EpeverConfiguration struct {
	SerialPort  string `yaml:"serialPort"`
	CacheExpiry int64  `yaml:"cacheExpiry"`
}

type EpeverController struct {
	handler *modbus.RTUClientHandler
	EpeverCollector *collector.EpeverCollector
	EpeverConfigurer *configurer.EpeverConfigurer
}

func NewEpeverController(config EpeverConfiguration) (*EpeverController, error) {
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

	epeverCollector := collector.NewEpeverCollector(client, config.CacheExpiry)
	epeverConfigurer := configurer.NewEpeverConfigurer(client)

	log.Infof("connected to epever %s", config.SerialPort)

	return &EpeverController{
		handler: handler,
		EpeverCollector: epeverCollector,
		EpeverConfigurer: epeverConfigurer,
	},nil
}

func (e *EpeverController) RegisterEndpoints(r *gin.Engine) {
	r.GET("/api/epever/metrics", e.EpeverCollector.MetricsGet())
	r.GET("/api/epever/config", e.EpeverConfigurer.ConfigGet())
	r.PATCH("/api/epever/config", e.EpeverConfigurer.ConfigPatch())
	r.POST("/api/epever/query", e.EpeverConfigurer.QueryPost())
}

func (e *EpeverController) Close() {
	e.handler.Close()
}
