package main

import (
	"flag"
	"fmt"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/lumberbarons/solar-controller/internal/controllers/epever"
	"github.com/lumberbarons/solar-controller/internal/controllers/pijuice"
	"github.com/lumberbarons/solar-controller/internal/controllers/victron"
	"github.com/lumberbarons/solar-controller/internal/publisher"
	staticfs "github.com/lumberbarons/solar-controller/internal/static"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
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
	HttpPort int                         `yaml:"httpPort"`
	Mqtt     publisher.MqttConfiguration `yaml:"mqtt"`
	Epever   epever.Configuration        `yaml:"epever"`
	Victron  victron.Configuration       `yaml:"victron"`
	Pijuice  pijuice.Configuration       `yaml:"pijuice"`
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
	r.SetTrustedProxies(nil)

	mqttPublisher, err := publisher.NewMqttPublisher(controllerConfig.SolarController.Mqtt)
	if err != nil {
		log.Fatalf("failed to create publisher: %v", err)
	}
	defer mqttPublisher.Close()

	controllers := buildControllers(controllerConfig, mqttPublisher)

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

	log.Infof("starting server on port %v", controllerConfig.SolarController.HttpPort)

	err = r.Run(fmt.Sprintf(":%v", controllerConfig.SolarController.HttpPort))
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func buildControllers(controllerConfig Config, mqttPublisher *publisher.MqttPublisher) []SolarController {
	var controllers []SolarController

	// epever

	epeverController, err := epever.NewController(controllerConfig.SolarController.Epever, mqttPublisher)
	if err != nil {
		log.Fatalf("failed to create epever controller: %v", err)
	}
	defer epeverController.Close()

	if epeverController.Enabled() {
		controllers = append(controllers, epeverController)
	}

	// victron

	victronController, err := victron.NewController(controllerConfig.SolarController.Victron, mqttPublisher)
	if err != nil {
		log.Fatalf("failed to create victron controller: %v", err)
	}
	defer victronController.Close()

	if victronController.Enabled() {
		controllers = append(controllers, victronController)
	}

	// pijuice

	pijuiceController, err := pijuice.NewController(controllerConfig.SolarController.Pijuice, mqttPublisher)
	if err != nil {
		log.Fatalf("failed to create pijuice controller: %v", err)
	}
	defer pijuiceController.Close()

	if pijuiceController.Enabled() {
		controllers = append(controllers, pijuiceController)
	}

	return controllers
}

func loadConfigFile() Config {
	if *configFilePath == "" {
		log.Fatalf("Must specify config file path")
	}

	configFile, err := ioutil.ReadFile(*configFilePath)
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
