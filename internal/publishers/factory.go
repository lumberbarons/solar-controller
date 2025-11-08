package publishers

import (
	"fmt"

	"github.com/lumberbarons/solar-controller/internal/config"
	"github.com/lumberbarons/solar-controller/internal/controllers"
	"github.com/lumberbarons/solar-controller/internal/mqtt"
	"github.com/lumberbarons/solar-controller/internal/solace"
	log "github.com/sirupsen/logrus"
)

// NewPublisher creates a message publisher based on the configuration.
// It returns an MQTT publisher if MQTT is enabled, a Solace publisher if Solace is enabled,
// or a no-op publisher if neither is enabled.
// Returns an error if both are enabled (should be caught by config validation) or if publisher creation fails.
func NewPublisher(cfg *config.SolarControllerConfiguration) (controllers.MessagePublisher, error) {
	// Check if both are enabled (should never happen due to config validation)
	if cfg.Mqtt.Enabled && cfg.Solace.Enabled {
		return nil, fmt.Errorf("both MQTT and Solace are enabled - only one publisher can be active")
	}

	// Try MQTT first
	if cfg.Mqtt.Enabled {
		log.Info("Creating MQTT publisher")
		return mqtt.NewPublisher(&cfg.Mqtt)
	}

	// Try Solace
	if cfg.Solace.Enabled {
		log.Info("Creating Solace publisher")
		return solace.NewPublisher(&cfg.Solace)
	}

	// Neither enabled - return a no-op publisher
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
