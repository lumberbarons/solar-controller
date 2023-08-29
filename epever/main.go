package epever

import (
	"github.com/goburrow/modbus"
	"github.com/lumberbarons/solar-controller/epever/collector"
	"github.com/lumberbarons/solar-controller/epever/configurer"
	log "github.com/sirupsen/logrus"
	"time"
)

type EpeverController struct {
	handler *modbus.RTUClientHandler
	EpeverCollector *collector.EpeverCollector
	EpeverConfigurer *configurer.EpeverConfigurer
}

func NewEpeverCollector(serialPort string, cacheExpiry int64) *EpeverController {
	handler := buildHandler(serialPort)
	client := modbus.NewClient(handler)

	epeverCollector := collector.NewEpeverCollector(client, cacheExpiry)
	epeverConfigurer := configurer.NewEpeverConfigurer(client)

	return &EpeverController{
		handler: handler,
		EpeverCollector: epeverCollector,
		EpeverConfigurer: epeverConfigurer,
	}
}

func (e *EpeverController) Close() {
	e.handler.Close()
}

func buildHandler(serialPort string) *modbus.RTUClientHandler {
	handler := modbus.NewRTUClientHandler(serialPort)

	handler.BaudRate = 115200
	handler.DataBits = 8
	handler.Parity = "N"
	handler.StopBits = 1
	handler.SlaveId = 1
	handler.Timeout = 2 * time.Second

	err := handler.Connect()

	if err != nil {
		log.Fatalf("Failed to connect to controller port: %v", err)
	}

	return handler
}
