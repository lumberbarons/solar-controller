package remotewrite

import (
	"fmt"
	"net/url"
	"time"
)

// Configuration holds the configuration for the Prometheus remote_write publisher.
type Configuration struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url"` // Required when enabled

	// Timeout for HTTP requests (default: 30s)
	Timeout string `yaml:"timeout"`

	// BasicAuth configuration (optional)
	BasicAuth *BasicAuthConfig `yaml:"basicAuth,omitempty"`

	// BearerToken for authentication (optional, mutually exclusive with BasicAuth)
	BearerToken string `yaml:"bearerToken,omitempty"`

	// Headers allows custom headers to be added to requests (optional)
	// Example: X-Scope-OrgID for Cortex multi-tenancy
	Headers map[string]string `yaml:"headers,omitempty"`
}

// BasicAuthConfig holds HTTP Basic Authentication credentials.
type BasicAuthConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Validate checks if the configuration is valid.
func (c *Configuration) Validate() error {
	if !c.Enabled {
		return nil
	}

	// URL is required when enabled
	if c.URL == "" {
		return fmt.Errorf("remoteWrite.url is required when enabled")
	}

	// Validate URL format
	parsedURL, err := url.Parse(c.URL)
	if err != nil {
		return fmt.Errorf("remoteWrite.url is invalid: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("remoteWrite.url must use http or https scheme")
	}

	// Validate timeout if specified
	if c.Timeout != "" {
		if _, err := time.ParseDuration(c.Timeout); err != nil {
			return fmt.Errorf("remoteWrite.timeout is invalid: %w", err)
		}
	}

	// BasicAuth and BearerToken are mutually exclusive
	if c.BasicAuth != nil && c.BearerToken != "" {
		return fmt.Errorf("remoteWrite.basicAuth and remoteWrite.bearerToken are mutually exclusive")
	}

	// If BasicAuth is provided, both username and password should be non-empty
	if c.BasicAuth != nil {
		if c.BasicAuth.Username == "" {
			return fmt.Errorf("remoteWrite.basicAuth.username is required when basicAuth is configured")
		}
		if c.BasicAuth.Password == "" {
			return fmt.Errorf("remoteWrite.basicAuth.password is required when basicAuth is configured")
		}
	}

	return nil
}

// GetTimeout returns the configured timeout or the default (30s).
func (c *Configuration) GetTimeout() time.Duration {
	if c.Timeout == "" {
		return 30 * time.Second
	}
	duration, err := time.ParseDuration(c.Timeout)
	if err != nil {
		// Should not happen if Validate() was called
		return 30 * time.Second
	}
	return duration
}
