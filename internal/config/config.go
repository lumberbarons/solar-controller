package config

import (
	"fmt"

	"github.com/lumberbarons/solar-controller/internal/controllers/epever"
	"github.com/lumberbarons/solar-controller/internal/publishers/file"
	"github.com/lumberbarons/solar-controller/internal/publishers/mqtt"
	"github.com/lumberbarons/solar-controller/internal/publishers/remotewrite"
	"github.com/lumberbarons/solar-controller/internal/publishers/sns"
	"github.com/lumberbarons/solar-controller/internal/publishers/solace"
	"gopkg.in/yaml.v3"
)

type Config struct {
	SolarController SolarControllerConfiguration `yaml:"solarController"`
}

type SolarControllerConfiguration struct {
	HTTPPort    int                       `yaml:"httpPort"`
	BindAddress string                    `yaml:"bindAddress"`
	Auth        AuthConfiguration         `yaml:"auth"`
	TLS         TLSConfiguration          `yaml:"tls"`
	Debug       bool                      `yaml:"debug"`
	DeviceID    string                    `yaml:"deviceId"`
	Mqtt        mqtt.Configuration        `yaml:"mqtt"`
	Solace      solace.Configuration      `yaml:"solace"`
	SNS         sns.Configuration         `yaml:"sns"`
	File        file.Configuration        `yaml:"file"`
	RemoteWrite remotewrite.Configuration `yaml:"remoteWrite"`
	Epever      epever.Configuration      `yaml:"epever"`
}

// AuthConfiguration holds API authentication settings. When Token is set,
// requests to /api routes must carry it as a bearer token.
type AuthConfiguration struct {
	Token string `yaml:"token"`
}

// TLSConfiguration holds the certificate pair for serving HTTPS. When both
// paths are set the HTTP server serves TLS; when both are empty it serves
// plain HTTP.
type TLSConfiguration struct {
	CertFile string `yaml:"certFile"`
	KeyFile  string `yaml:"keyFile"`
}

// Enabled reports whether a TLS certificate pair is configured.
func (t TLSConfiguration) Enabled() bool {
	return t.CertFile != "" && t.KeyFile != ""
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
	// Bind to loopback unless the operator explicitly exposes the server
	if c.SolarController.BindAddress == "" {
		c.SolarController.BindAddress = "127.0.0.1"
	}

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
	if c.SolarController.SNS.TopicPrefix == "" {
		c.SolarController.SNS.TopicPrefix = "solar"
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

	// TLS requires both halves of the certificate pair
	tls := c.SolarController.TLS
	if (tls.CertFile == "") != (tls.KeyFile == "") {
		return fmt.Errorf("tls.certFile and tls.keyFile must both be set to enable TLS")
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

	// Validate SNS configuration if enabled
	if c.SolarController.SNS.Enabled {
		if c.SolarController.SNS.TopicArn == "" {
			return fmt.Errorf("SNS topic ARN is required when SNS is enabled")
		}
		if c.SolarController.SNS.Region == "" {
			return fmt.Errorf("SNS region is required when SNS is enabled")
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
