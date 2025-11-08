package solace

import (
	"fmt"
	"time"

	"solace.dev/go/messaging"
	"solace.dev/go/messaging/pkg/solace"
	"solace.dev/go/messaging/pkg/solace/config"
	"solace.dev/go/messaging/pkg/solace/resource"

	log "github.com/sirupsen/logrus"
)

type Configuration struct {
	Enabled  bool   `yaml:"enabled"`
	Host     string `yaml:"host"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	VpnName  string `yaml:"vpnName"`
}

type Publisher struct {
	messagingService solace.MessagingService
	publisher        solace.DirectMessagePublisher
	config           Configuration
	topicPrefix      string
}

func NewPublisher(cfg *Configuration, topicPrefix string) (*Publisher, error) {
	if !cfg.Enabled {
		log.Info("Solace publisher disabled via configuration")
		return &Publisher{}, nil
	}

	if cfg.Host == "" {
		log.Warn("Solace enabled but no host provided, publisher disabled")
		return &Publisher{}, nil
	}

	if cfg.VpnName == "" {
		log.Warn("Solace enabled but no VPN name provided, publisher disabled")
		return &Publisher{}, nil
	}

	// Build messaging service configuration
	brokerConfig := cfg.ServicePropertyMap(cfg.Host, cfg.VpnName, cfg.Username, cfg.Password)

	// Build the messaging service
	messagingService, err := messaging.NewMessagingServiceBuilder().
		FromConfigurationProvider(brokerConfig).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build messaging service: %w", err)
	}

	// Connect to the messaging service
	if err := messagingService.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to Solace broker: %w", err)
	}

	log.Infof("connected to Solace broker %s (VPN: %s)", cfg.Host, cfg.VpnName)

	// Build a direct message publisher
	directPublisher, err := messagingService.CreateDirectMessagePublisherBuilder().Build()
	if err != nil {
		if disconnectErr := messagingService.Disconnect(); disconnectErr != nil {
			log.Warnf("failed to disconnect messaging service during cleanup: %v", disconnectErr)
		}
		return nil, fmt.Errorf("failed to create direct message publisher: %w", err)
	}

	// Start the publisher
	if err := directPublisher.Start(); err != nil {
		if disconnectErr := messagingService.Disconnect(); disconnectErr != nil {
			log.Warnf("failed to disconnect messaging service during cleanup: %v", disconnectErr)
		}
		return nil, fmt.Errorf("failed to start direct message publisher: %w", err)
	}

	publisher := &Publisher{
		messagingService: messagingService,
		publisher:        directPublisher,
		config:           *cfg,
		topicPrefix:      topicPrefix,
	}

	return publisher, nil
}

func (p *Publisher) Publish(topicSuffix, payload string) {
	if p.publisher == nil {
		return
	}

	topic := fmt.Sprintf("%s/%s", p.topicPrefix, topicSuffix)

	log.Infof("publishing for %s to %s", topicSuffix, topic)

	// Create a topic destination
	destination := resource.TopicOf(topic)

	// Create a message with the payload
	message, err := p.messagingService.MessageBuilder().
		BuildWithStringPayload(payload)
	if err != nil {
		log.Errorf("failed to build message for %s: %s", topicSuffix, err)
		return
	}

	// Publish the message with a timeout
	// Using a channel to implement timeout for fire-and-forget publish
	done := make(chan error, 1)
	go func() {
		done <- p.publisher.Publish(message, destination)
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Errorf("failed to publish to %s: %s", topic, err)
		}
	case <-time.After(5 * time.Second):
		log.Errorf("timeout waiting for publish for %s collector", topicSuffix)
	}
}

func (p *Publisher) Close() {
	if p.publisher != nil {
		if err := p.publisher.Terminate(250 * time.Millisecond); err != nil {
			log.Warnf("failed to terminate publisher: %v", err)
		}
	}
	if p.messagingService != nil {
		if err := p.messagingService.Disconnect(); err != nil {
			log.Warnf("failed to disconnect messaging service: %v", err)
		}
	}
}

// ServicePropertyMap creates a configuration provider for the Solace messaging service
func (c *Configuration) ServicePropertyMap(host, vpnName, username, password string) config.ServicePropertyMap {
	return config.ServicePropertyMap{
		config.TransportLayerPropertyHost:                host,
		config.ServicePropertyVPNName:                    vpnName,
		config.AuthenticationPropertySchemeBasicUserName: username,
		config.AuthenticationPropertySchemeBasicPassword: password,
	}
}
