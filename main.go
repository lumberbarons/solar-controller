package main

import (
	"github.com/gin-gonic/gin"
	"github.com/goburrow/modbus"
	"github.com/lumberbarons/epever_controller/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"time"
)

func prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

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
	prometheus.MustRegister(metrics.NewMetricsCollector(client))

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	r.GET("/metrics", prometheusHandler())

	r.Run()
}