package main

import (
	"flag"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/lumberbarons/solar-controller/internal/app"
	"github.com/lumberbarons/solar-controller/internal/config"
	"github.com/lumberbarons/solar-controller/internal/publishers"
	log "github.com/sirupsen/logrus"
)

var (
	configFilePath *string
	debugMode      *bool

	// Version information injected at build time via ldflags
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
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
	log.Infof("starting solar-controller version %s (commit: %s, built: %s)", Version, GitCommit, BuildTime)

	flag.Parse()

	controllerConfig := loadConfigFile()

	// Command line flag takes precedence over config file
	debugEnabled := *debugMode || controllerConfig.SolarController.Debug

	if debugEnabled {
		log.SetLevel(log.DebugLevel)
		log.Debug("debug mode enabled")
	} else {
		log.SetLevel(log.InfoLevel)
		gin.SetMode(gin.ReleaseMode)
	}

	publisher, err := publishers.NewPublisher(&controllerConfig.SolarController)
	if err != nil {
		log.Fatalf("failed to create publisher: %v", err)
	}

	versionInfo := app.VersionInfo{
		Version:   Version,
		BuildTime: BuildTime,
		GitCommit: GitCommit,
	}

	application, err := app.NewApplication(&controllerConfig, publisher, versionInfo)
	if err != nil {
		log.Fatalf("failed to create application: %v", err)
	}
	defer func() {
		if err := application.Close(); err != nil {
			log.Errorf("failed to close application: %v", err)
		}
	}()

	if err := application.Run(); err != nil {
		log.Errorf("failed to start server: %v", err)
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
