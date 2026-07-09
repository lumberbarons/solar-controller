package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// Gin always runs in release mode: the debug flag controls application
	// log verbosity only, not Gin's verbose route dumps and stack traces
	gin.SetMode(gin.ReleaseMode)

	// Command line flag takes precedence over config file
	debugEnabled := *debugMode || controllerConfig.SolarController.Debug

	if debugEnabled {
		log.SetLevel(log.DebugLevel)
		log.Debug("debug mode enabled")
	} else {
		log.SetLevel(log.InfoLevel)
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

	// On SIGINT/SIGTERM, drain in-flight HTTP requests (including Modbus
	// EEPROM writes) before the process exits
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		log.Info("shutdown signal received, draining connections")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := application.Shutdown(shutdownCtx); err != nil {
			log.Errorf("graceful shutdown failed: %v", err)
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
