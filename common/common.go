package common

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

type SolarController interface {
	GetSolarCollector() SolarCollector
	GetPrometheusCollector() prometheus.Collector
	RegisterEndpoints(r *gin.Engine)
}

type SolarCollector interface {
	GetTopicSuffix() string
	GetStatusString() (string, error)
}
