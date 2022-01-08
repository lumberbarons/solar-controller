package main

import (
	"github.com/gin-gonic/gin"
	"github.com/lumberbarons/epever_controller/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func main() {
	log.Println("starting epever-controller")

	prometheus.MustRegister(metrics.NewMetricsCollector("/dev/ttys003"))

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	r.GET("/metrics", prometheusHandler())

	r.Run()
}