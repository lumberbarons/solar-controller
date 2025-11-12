package config

import (
	"fmt"

	"github.com/lumberbarons/solar-controller/internal/controllers/epever"
	"github.com/lumberbarons/solar-controller/internal/file"
	"github.com/lumberbarons/solar-controller/internal/mqtt"
	"github.com/lumberbarons/solar-controller/internal/remotewrite"
	"github.com/lumberbarons/solar-controller/internal/solace"
	"gopkg.in/yaml.v3"
)

type Config struct {
	SolarController SolarControllerConfiguration `yaml:"solarController"`
}

type SolarControllerConfiguration struct {
	HTTPPort    int                       `yaml:"httpPort"`
	Debug       bool                      `yaml:"debug"`
	DeviceID    string                    `yaml:"deviceId"`
	Mqtt        mqtt.Configuration        `yaml:"mqtt"`
	Solace      solace.Configuration      `yaml:"solace"`
	File        file.Configuration        `yaml:"file"`
	RemoteWrite remotewrite.Configuration `yaml:"remoteWrite"`
	Epever      epever.Configuration      `yaml:"epever"`
}

// Load parses and validates configuration from YAML bytes.
// This is a pure function for testing - it doesn't read files or exit the process.
func Load(data []byte) (Config, error) {
	var config Config

	err := yaml.Unmarshal(data, &config)
	if err != nil {
		return Config{}, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Apply defaults
	config.applyDefaults()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// applyDefaults sets default values for optional configuration fields
func (c *Config) applyDefaults() {
	// Set default device ID if not provided
	if c.SolarController.DeviceID == "" {
		c.SolarController.DeviceID = "controller-1"
	}

	// Set default topic prefix for publishers that use it
	if c.SolarController.Mqtt.TopicPrefix == "" {
		c.SolarController.Mqtt.TopicPrefix = "solar"
	}
	if c.SolarController.Solace.TopicPrefix == "" {
		c.SolarController.Solace.TopicPrefix = "solar"
	}
	if c.SolarController.RemoteWrite.TopicPrefix == "" {
		c.SolarController.RemoteWrite.TopicPrefix = "solar"
	}
	// Note: File publisher does not use topicPrefix
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.SolarController.HTTPPort <= 0 || c.SolarController.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d (must be 1-65535)", c.SolarController.HTTPPort)
	}

	// Note: Multiple publishers can now be enabled simultaneously

	// Validate MQTT configuration if enabled
	if c.SolarController.Mqtt.Enabled {
		if c.SolarController.Mqtt.Host == "" {
			return fmt.Errorf("MQTT host is required when MQTT is enabled")
		}
	}

	// Validate Solace configuration if enabled
	if c.SolarController.Solace.Enabled {
		if c.SolarController.Solace.Host == "" {
			return fmt.Errorf("solace host is required when Solace is enabled")
		}
		if c.SolarController.Solace.VpnName == "" {
			return fmt.Errorf("solace VPN name is required when Solace is enabled")
		}
	}

	// Validate File configuration if enabled
	if c.SolarController.File.Enabled {
		if c.SolarController.File.Filename == "" {
			return fmt.Errorf("file filename is required when File publisher is enabled")
		}
	}

	// Validate RemoteWrite configuration if enabled
	if c.SolarController.RemoteWrite.Enabled {
		if err := c.SolarController.RemoteWrite.Validate(); err != nil {
			return err
		}
	}

	// Validate Epever configuration if enabled
	if c.SolarController.Epever.Enabled {
		if c.SolarController.Epever.SerialPort == "" {
			return fmt.Errorf("epever serial port is required when epever is enabled")
		}
		if c.SolarController.Epever.PublishPeriod <= 0 {
			return fmt.Errorf("epever publish period must be positive")
		}
	}

	return nil
}
