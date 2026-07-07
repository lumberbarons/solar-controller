package publishers

import (
	"fmt"

	"github.com/lumberbarons/solar-controller/internal/config"
	"github.com/lumberbarons/solar-controller/internal/publish"
	"github.com/lumberbarons/solar-controller/internal/publishers/file"
	"github.com/lumberbarons/solar-controller/internal/publishers/mqtt"
	"github.com/lumberbarons/solar-controller/internal/publishers/remotewrite"
	"github.com/lumberbarons/solar-controller/internal/publishers/sns"
	"github.com/lumberbarons/solar-controller/internal/publishers/solace"
	log "github.com/sirupsen/logrus"
)

// NewPublisher creates a message publisher based on the configuration.
// It returns a MultiPublisher wrapping all enabled publishers, or a no-op publisher if none is enabled.
// Returns an error if publisher creation fails.
func NewPublisher(cfg *config.SolarControllerConfiguration) (publish.MessagePublisher, error) {
	var publishers []publish.MessagePublisher

	// Create MQTT publisher if enabled
	if cfg.Mqtt.Enabled {
		log.Info("Creating MQTT publisher")
		publisher, err := mqtt.NewPublisher(&cfg.Mqtt)
		if err != nil {
			return nil, fmt.Errorf("failed to create MQTT publisher: %w", err)
		}
		publishers = append(publishers, publisher)
	}

	// Create Solace publisher if enabled
	if cfg.Solace.Enabled {
		log.Info("Creating Solace publisher")
		publisher, err := solace.NewPublisher(&cfg.Solace)
		if err != nil {
			return nil, fmt.Errorf("failed to create Solace publisher: %w", err)
		}
		publishers = append(publishers, publisher)
	}

	// Create SNS publisher if enabled
	if cfg.SNS.Enabled {
		log.Info("Creating SNS publisher")
		publisher, err := sns.NewPublisher(&cfg.SNS)
		if err != nil {
			return nil, fmt.Errorf("failed to create SNS publisher: %w", err)
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
		publisher, err := remotewrite.NewPublisher(&cfg.RemoteWrite, cfg.DeviceID)
		if err != nil {
			return nil, fmt.Errorf("failed to create RemoteWrite publisher: %w", err)
		}
		publishers = append(publishers, publisher)
	}

	// If no publishers enabled, return no-op publisher
	if len(publishers) == 0 {
		log.Info("No message publisher enabled")
		return &publish.NoOpPublisher{}, nil
	}

	// If only one publisher, return it directly (no need for MultiPublisher wrapper)
	if len(publishers) == 1 {
		return publishers[0], nil
	}

	// Multiple publishers - wrap in MultiPublisher
	log.Infof("Creating MultiPublisher with %d publishers", len(publishers))
	return NewMultiPublisher(publishers, log.StandardLogger()), nil
}
