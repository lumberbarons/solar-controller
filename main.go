package main

import (
	"github.com/gin-gonic/gin"
	"github.com/goburrow/modbus"
	"github.com/lumberbarons/epever_controller/config"
	"github.com/lumberbarons/epever_controller/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

func init() {
	if os.Getenv("DEBUG") == "true" {
		log.SetLevel(log.DebugLevel)
	}
}

func main() {
	log.Println("starting epever-controller")

	serialPort := os.Getenv("SERIAL_PORT")
	if serialPort == "" {
		log.Fatal("serial port not set")
	}

	handler := modbus.NewRTUClientHandler(serialPort)

	handler.BaudRate = 115200
	handler.DataBits = 8
	handler.Parity = "N"
	handler.StopBits = 1
	handler.SlaveId = 1
	handler.Timeout = 5 * time.Second

	err := handler.Connect()
	if err != nil {
		log.Fatal("failed to connect to serial port")
	}

	defer handler.Close()

	client := modbus.NewClient(handler)
	prometheus.MustRegister()

	r := gin.Default()

	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.NewSolarCollector(client))
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	r.GET("/metrics", func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	})

	configurer := config.NewSolarConfigurer(client)

	r.GET("/api/time", configurer.TimeGet())
	r.PUT("/api/time", configurer.TimePut())

	r.Run()
}