package publishers

import (
	"fmt"

	"github.com/lumberbarons/solar-controller/internal/config"
	"github.com/lumberbarons/solar-controller/internal/controllers"
	"github.com/lumberbarons/solar-controller/internal/file"
	"github.com/lumberbarons/solar-controller/internal/mqtt"
	"github.com/lumberbarons/solar-controller/internal/solace"
	log "github.com/sirupsen/logrus"
)

// NewPublisher creates a message publisher based on the configuration.
// It returns an MQTT publisher if MQTT is enabled, a Solace publisher if Solace is enabled,
// a File publisher if File is enabled, or a no-op publisher if none is enabled.
// Returns an error if multiple publishers are enabled (should be caught by config validation) or if publisher creation fails.
func NewPublisher(cfg *config.SolarControllerConfiguration) (controllers.MessagePublisher, error) {
	// Count enabled publishers (should never be > 1 due to config validation)
	enabledCount := 0
	if cfg.Mqtt.Enabled {
		enabledCount++
	}
	if cfg.Solace.Enabled {
		enabledCount++
	}
	if cfg.File.Enabled {
		enabledCount++
	}

	if enabledCount > 1 {
		return nil, fmt.Errorf("multiple publishers are enabled - only one publisher can be active")
	}

	// Try MQTT first
	if cfg.Mqtt.Enabled {
		log.Info("Creating MQTT publisher")
		return mqtt.NewPublisher(&cfg.Mqtt, cfg.TopicPrefix)
	}

	// Try Solace
	if cfg.Solace.Enabled {
		log.Info("Creating Solace publisher")
		return solace.NewPublisher(&cfg.Solace, cfg.TopicPrefix)
	}

	// Try File
	if cfg.File.Enabled {
		log.Info("Creating File publisher")
		return file.NewPublisher(&cfg.File, cfg.TopicPrefix)
	}

	// None enabled - return a no-op publisher
	log.Info("No message publisher enabled")
	return &NoOpPublisher{}, nil
}

// NoOpPublisher is a publisher that does nothing.
// Used when neither MQTT nor Solace is configured.
type NoOpPublisher struct{}

func (n *NoOpPublisher) Publish(_, _ string) {
	// Do nothing
}

func (n *NoOpPublisher) Close() {
	// Do nothing
}
