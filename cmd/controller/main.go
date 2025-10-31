package main

import (
	"flag"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/lumberbarons/solar-controller/internal/app"
	"github.com/lumberbarons/solar-controller/internal/config"
	"github.com/lumberbarons/solar-controller/internal/mqtt"
	log "github.com/sirupsen/logrus"
)

var (
	configFilePath *string
	debugMode      *bool
)

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

	mqttPublisher, err := mqtt.NewPublisher(controllerConfig.SolarController.Mqtt)
	if err != nil {
		log.Fatalf("failed to create publisher: %v", err)
	}

	application, err := app.NewApplication(&controllerConfig, mqttPublisher)
	if err != nil {
		log.Fatalf("failed to create application: %v", err)
	}
	defer application.Close()

	if err := application.Run(); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func loadConfigFile() config.Config {
	if *configFilePath == "" {
		log.Fatalf("Must specify config file path")
	}

	configFile, err := os.ReadFile(*configFilePath)
	if err != nil {
		log.Fatalf("failed to load configurer file: %v", err)
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	return cfg
}
