package app

import (
	"fmt"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/lumberbarons/solar-controller/internal/config"
	"github.com/lumberbarons/solar-controller/internal/controllers"
	"github.com/lumberbarons/solar-controller/internal/controllers/epever"
	staticfs "github.com/lumberbarons/solar-controller/internal/static"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

// Application encapsulates the solar-controller application, including
// the HTTP server, MQTT publisher, and solar equipment controllers.
type Application struct {
	config      *config.Config
	router      *gin.Engine
	mqtt        controllers.MessagePublisher
	controllers []controllers.SolarController
}

// NewApplication creates and initializes a new Application instance.
// It sets up the HTTP router, initializes controllers, and registers endpoints.
func NewApplication(cfg *config.Config, mqttPublisher controllers.MessagePublisher) (*Application, error) {
	app := &Application{
		config: cfg,
		mqtt:   mqttPublisher,
	}

	// Initialize router
	app.router = gin.Default()
	if err := app.router.SetTrustedProxies(nil); err != nil {
		log.Warnf("failed to set trusted proxies: %v", err)
	}

	// Build controllers
	if err := app.buildControllers(); err != nil {
		return nil, fmt.Errorf("failed to build controllers: %w", err)
	}

	// Setup routes
	app.setupRoutes()

	return app, nil
}

// buildControllers initializes all solar equipment controllers based on configuration.
func (a *Application) buildControllers() error {
	var ctrlList []controllers.SolarController

	// Initialize Epever controller
	epeverController, err := epever.NewControllerFromConfig(a.config.SolarController.Epever, a.mqtt)
	if err != nil {
		return fmt.Errorf("failed to create epever controller: %w", err)
	}

	if epeverController.Enabled() {
		ctrlList = append(ctrlList, epeverController)
	}

	a.controllers = ctrlList
	return nil
}

// setupRoutes configures all HTTP routes for the application.
func (a *Application) setupRoutes() {
	// Prometheus metrics endpoint
	handler := promhttp.Handler()
	a.router.GET("/metrics", func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	})

	// Register controller-specific endpoints
	for _, controller := range a.controllers {
		if controller.Enabled() {
			controller.RegisterEndpoints(a.router)
		}
	}

	// Serve static frontend (React app)
	siteFS := staticfs.GetSiteFS()
	a.router.Use(static.Serve("/", siteFS))
}

// Run starts the HTTP server and blocks until it exits.
func (a *Application) Run() error {
	log.Infof("starting server on port %v", a.config.SolarController.HTTPPort)
	return a.router.Run(fmt.Sprintf(":%v", a.config.SolarController.HTTPPort))
}

// Close performs cleanup of all application resources.
// It closes the MQTT publisher and all controllers.
func (a *Application) Close() error {
	log.Info("shutting down application")

	// Close MQTT publisher
	if a.mqtt != nil {
		a.mqtt.Close()
	}

	// Close all controllers
	for _, controller := range a.controllers {
		if err := controller.Close(); err != nil {
			log.Errorf("failed to close controller: %v", err)
		}
	}

	return nil
}

// Router returns the Gin router instance for testing purposes.
func (a *Application) Router() *gin.Engine {
	return a.router
}
