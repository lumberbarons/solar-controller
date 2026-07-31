package main

import (
	"context"
	"flag"
	"fmt"
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

// main handles only flag parsing and turning a startup failure into a non-zero
// exit. Everything it does beyond that lives in functions that return errors, so
// startup can be exercised by tests rather than terminating the process.
func main() {
	flag.Parse()

	if err := run(*configFilePath, *debugMode); err != nil {
		log.Fatal(err)
	}
}

// run starts the application and blocks until the HTTP server exits.
func run(configPath string, debugFlag bool) error {
	log.Infof("starting solar-controller version %s (commit: %s, built: %s)", Version, GitCommit, BuildTime)

	controllerConfig, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	// Gin always runs in release mode: the debug flag controls application
	// log verbosity only, not Gin's verbose route dumps and stack traces
	gin.SetMode(gin.ReleaseMode)

	configureLogging(resolveDebugMode(debugFlag, &controllerConfig))

	publisher, err := publishers.NewPublisher(&controllerConfig.SolarController)
	if err != nil {
		return fmt.Errorf("failed to create publisher: %w", err)
	}

	application, err := app.NewApplication(&controllerConfig, publisher, versionInfo())
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
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
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// versionInfo returns the build metadata injected via ldflags.
func versionInfo() app.VersionInfo {
	return app.VersionInfo{
		Version:   Version,
		BuildTime: BuildTime,
		GitCommit: GitCommit,
	}
}

// resolveDebugMode reports whether debug logging is on. The command line flag
// takes precedence over the config file, so -debug can enable it for a config
// that does not ask for it.
func resolveDebugMode(debugFlag bool, cfg *config.Config) bool {
	return debugFlag || cfg.SolarController.Debug
}

func configureLogging(debugEnabled bool) {
	if debugEnabled {
		log.SetLevel(log.DebugLevel)
		log.Debug("debug mode enabled")
		return
	}
	log.SetLevel(log.InfoLevel)
}

// loadConfig reads and parses the config file at path.
func loadConfig(path string) (config.Config, error) {
	if path == "" {
		return config.Config{}, fmt.Errorf("must specify config file path")
	}

	configFile, err := os.ReadFile(path)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to load configuration: %w", err)
	}

	return cfg, nil
}
