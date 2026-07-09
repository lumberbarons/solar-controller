package app

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"time"

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
	version     VersionInfo
}

// VersionInfo holds version metadata about the application.
type VersionInfo struct {
	Version   string `json:"version"`
	BuildTime string `json:"buildTime"`
	GitCommit string `json:"gitCommit"`
}

// NewApplication creates and initializes a new Application instance.
// It sets up the HTTP router, initializes controllers, and registers endpoints.
func NewApplication(cfg *config.Config, mqttPublisher controllers.MessagePublisher, version VersionInfo) (*Application, error) {
	app := &Application{
		config:  cfg,
		mqtt:    mqttPublisher,
		version: version,
	}

	// Initialize router
	app.router = gin.Default()
	if err := app.router.SetTrustedProxies(nil); err != nil {
		log.Warnf("failed to set trusted proxies: %v", err)
	}

	if cfg.SolarController.Auth.Token != "" {
		app.router.Use(authMiddleware(cfg.SolarController.Auth.Token))
	} else {
		log.Warn("no auth token configured; /api endpoints are unauthenticated")
	}

	// Build controllers
	if err := app.buildControllers(); err != nil {
		return nil, fmt.Errorf("failed to build controllers: %w", err)
	}

	// Setup routes
	app.setupRoutes()

	return app, nil
}

// authMiddleware requires a bearer token on all /api routes. The SPA and
// static assets remain public so the frontend can load and prompt for a token.
func authMiddleware(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.Next()
			return
		}
		expected := "Bearer " + token
		provided := c.GetHeader("Authorization")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

// buildControllers initializes all solar equipment controllers based on configuration.
func (a *Application) buildControllers() error {
	var ctrlList []controllers.SolarController

	// Initialize Epever controller
	epeverController, err := epever.NewControllerFromConfig(a.config.SolarController.Epever, a.mqtt, a.config.SolarController.DeviceID)
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

	// Version info endpoint
	a.router.GET("/api/info", func(c *gin.Context) {
		c.JSON(200, a.version)
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

	// SPA fallback: serve index.html for any route that doesn't match
	// This allows React Router to handle client-side routing
	a.router.NoRoute(func(c *gin.Context) {
		c.FileFromFS("/", siteFS)
	})
}

// newHTTPServer builds the http.Server used by Run. Timeouts are set
// explicitly so slow clients cannot hold connections open indefinitely;
// the generous read/write timeouts leave room for PATCH handlers that
// perform multi-second Modbus writes.
func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    64 * 1024,
	}
}

// Run starts the HTTP server and blocks until it exits. When a TLS
// certificate pair is configured the server serves HTTPS instead.
func (a *Application) Run() error {
	addr := fmt.Sprintf("%s:%v", a.config.SolarController.BindAddress, a.config.SolarController.HTTPPort)
	srv := newHTTPServer(addr, a.router)

	if tls := a.config.SolarController.TLS; tls.Enabled() {
		log.Infof("starting HTTPS server on %s", addr)
		return srv.ListenAndServeTLS(tls.CertFile, tls.KeyFile)
	}

	log.Infof("starting server on %s", addr)
	return srv.ListenAndServe()
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
