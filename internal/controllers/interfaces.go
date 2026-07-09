package controllers

import (
	"github.com/gin-gonic/gin"
)

// SolarController defines the interface that all solar equipment controllers must implement.
// Controllers manage the lifecycle of hardware communication, metrics collection, and API endpoints.
type SolarController interface {
	// RegisterEndpoints registers HTTP endpoints for this controller.
	RegisterEndpoints(r *gin.Engine)

	// Enabled returns whether this controller is enabled and should be started.
	Enabled() bool

	// Close performs cleanup and releases resources held by the controller.
	Close() error
}
