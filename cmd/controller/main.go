package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/lumberbarons/solar-controller/internal/controllers/epever"
	"github.com/lumberbarons/solar-controller/internal/controllers/victron"
	"github.com/lumberbarons/solar-controller/internal/mqtt"
	staticfs "github.com/lumberbarons/solar-controller/internal/static"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

var (
	configFilePath *string
	debugMode      *bool
)

type SolarController interface {
	RegisterEndpoints(r *gin.Engine)
	Enabled() bool
}

type Config struct {
	SolarController SolarControllerConfiguration `yaml:"solarController"`
}

type SolarControllerConfiguration struct {
	HTTPPort int                   `yaml:"httpPort"`
	Mqtt     mqtt.Configuration    `yaml:"mqtt"`
	Epever   epever.Configuration  `yaml:"epever"`
	Victron  victron.Configuration `yaml:"victron"`
}

func init() {
	configFilePath = flag.String("config", "", "Config file path")
	debugMode = flag.Bool("debug", false, "Debug mode")

	log.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})
}

func main() {
	log.Info("starting solar-controller")

	flag.Parse()

	if *debugMode {
		log.SetLevel(log.DebugLevel)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	controllerConfig := loadConfigFile()

	r := gin.Default()
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Warnf("failed to set trusted proxies: %v", err)
	}

	mqttPublisher, err := mqtt.NewPublisher(controllerConfig.SolarController.Mqtt)
	if err != nil {
		log.Fatalf("failed to create publisher: %v", err)
	}

	controllers := buildControllers(&controllerConfig, mqttPublisher)

	handler := promhttp.Handler()
	r.GET("/metrics", func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	})

	for _, controller := range controllers {
		if controller.Enabled() {
			controller.RegisterEndpoints(r)
		}
	}

	siteFS := staticfs.GetSiteFS()
	r.Use(static.Serve("/", siteFS))

	log.Infof("starting server on port %v", controllerConfig.SolarController.HTTPPort)

	err = r.Run(fmt.Sprintf(":%v", controllerConfig.SolarController.HTTPPort))

	// Cleanup on exit
	mqttPublisher.Close()
	for _, controller := range controllers {
		if epeverCtrl, ok := controller.(*epever.Controller); ok {
			epeverCtrl.Close()
		}
		if victronCtrl, ok := controller.(*victron.Controller); ok {
			victronCtrl.Close()
		}
	}

	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func buildControllers(controllerConfig *Config, mqttPublisher *mqtt.Publisher) []SolarController {
	var controllers []SolarController

	// epever

	epeverController, err := epever.NewController(controllerConfig.SolarController.Epever, mqttPublisher)
	if err != nil {
		log.Fatalf("failed to create epever controller: %v", err)
	}

	if epeverController.Enabled() {
		controllers = append(controllers, epeverController)
	}

	// victron

	victronController, err := victron.NewController(controllerConfig.SolarController.Victron, mqttPublisher)
	if err != nil {
		log.Fatalf("failed to create victron controller: %v", err)
	}

	if victronController.Enabled() {
		controllers = append(controllers, victronController)
	}

	return controllers
}

func loadConfigFile() Config {
	if *configFilePath == "" {
		log.Fatalf("Must specify config file path")
	}

	configFile, err := os.ReadFile(*configFilePath)
	if err != nil {
		log.Fatalf("failed to load configurer file: %v", err)
	}

	config := Config{}

	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		log.Fatal(err)
	}

	return config
}
