package voltgo

import (
	"time"
)

const (
	namespace = "voltgo"

	defaultConnectTimeout = 30 * time.Second
)

type Configuration struct {
	Enabled       bool   `yaml:"enabled"`
	Address       string `yaml:"address"`
	PublishPeriod int    `yaml:"publishPeriod"`

	// ConnectTimeout is the maximum time to wait for a BLE connection,
	// as a duration string (default: 30s)
	ConnectTimeout string `yaml:"connectTimeout"`
}

// Validate checks the configuration for errors. Only called when enabled.
func (c *Configuration) Validate() error {
	if c.ConnectTimeout != "" {
		if _, err := time.ParseDuration(c.ConnectTimeout); err != nil {
			return err
		}
	}
	return nil
}

// GetConnectTimeout returns the configured connect timeout or the default (30s).
func (c *Configuration) GetConnectTimeout() time.Duration {
	if c.ConnectTimeout == "" {
		return defaultConnectTimeout
	}
	timeout, err := time.ParseDuration(c.ConnectTimeout)
	if err != nil {
		return defaultConnectTimeout
	}
	return timeout
}
