package publishers

import (
	"fmt"

	"github.com/lumberbarons/solar-controller/internal/config"
	"github.com/lumberbarons/solar-controller/internal/controllers"
	"github.com/lumberbarons/solar-controller/internal/file"
	"github.com/lumberbarons/solar-controller/internal/mqtt"
	"github.com/lumberbarons/solar-controller/internal/remotewrite"
	"github.com/lumberbarons/solar-controller/internal/solace"
	log "github.com/sirupsen/logrus"
)

// NewPublisher creates a message publisher based on the configuration.
// It returns a MultiPublisher wrapping all enabled publishers, or a no-op publisher if none is enabled.
// Returns an error if publisher creation fails.
func NewPublisher(cfg *config.SolarControllerConfiguration) (controllers.MessagePublisher, error) {
	var publishers []controllers.MessagePublisher

	// Create MQTT publisher if enabled
	if cfg.Mqtt.Enabled {
		log.Info("Creating MQTT publisher")
		publisher, err := mqtt.NewPublisher(&cfg.Mqtt, cfg.Mqtt.TopicPrefix)
		if err != nil {
			return nil, fmt.Errorf("failed to create MQTT publisher: %w", err)
		}
		publishers = append(publishers, publisher)
	}

	// Create Solace publisher if enabled
	if cfg.Solace.Enabled {
		log.Info("Creating Solace publisher")
		publisher, err := solace.NewPublisher(&cfg.Solace, cfg.Solace.TopicPrefix)
		if err != nil {
			return nil, fmt.Errorf("failed to create Solace publisher: %w", err)
		}
		publishers = append(publishers, publisher)
	}

	// Create File publisher if enabled
	if cfg.File.Enabled {
		log.Info("Creating File publisher")
		publisher, err := file.NewPublisher(&cfg.File)
		if err != nil {
			return nil, fmt.Errorf("failed to create File publisher: %w", err)
		}
		publishers = append(publishers, publisher)
	}

	// Create RemoteWrite publisher if enabled
	if cfg.RemoteWrite.Enabled {
		log.Info("Creating Prometheus RemoteWrite publisher")
		publisher, err := remotewrite.NewPublisher(&cfg.RemoteWrite, cfg.RemoteWrite.TopicPrefix, cfg.DeviceID)
		if err != nil {
			return nil, fmt.Errorf("failed to create RemoteWrite publisher: %w", err)
		}
		publishers = append(publishers, publisher)
	}

	// If no publishers enabled, return no-op publisher
	if len(publishers) == 0 {
		log.Info("No message publisher enabled")
		return &NoOpPublisher{}, nil
	}

	// If only one publisher, return it directly (no need for MultiPublisher wrapper)
	if len(publishers) == 1 {
		return publishers[0], nil
	}

	// Multiple publishers - wrap in MultiPublisher
	log.Infof("Creating MultiPublisher with %d publishers", len(publishers))
	return NewMultiPublisher(publishers, log.StandardLogger()), nil
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
