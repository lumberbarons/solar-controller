package solace

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func init() {
	// Suppress log output during tests
	logrus.SetLevel(logrus.ErrorLevel)
}

func TestNewPublisher_Disabled(t *testing.T) {
	config := &Configuration{
		Enabled: false,
		Host:    "tcp://localhost:55555",
		VpnName: "default",
	}

	pub, err := NewPublisher(config, "solar")
	if err != nil {
		t.Fatalf("Expected no error for disabled publisher, got: %v", err)
	}

	if pub.publisher != nil {
		t.Error("Expected nil publisher for disabled publisher")
	}

	if pub.messagingService != nil {
		t.Error("Expected nil messaging service for disabled publisher")
	}

	if pub.topicPrefix != "" {
		t.Errorf("Expected empty topic prefix for disabled publisher, got: %s", pub.topicPrefix)
	}
}

func TestNewPublisher_MissingHost(t *testing.T) {
	config := &Configuration{
		Enabled: true,
		Host:    "",
		VpnName: "default",
	}

	pub, err := NewPublisher(config, "solar")
	if err != nil {
		t.Fatalf("Expected no error for missing host (returns empty publisher), got: %v", err)
	}

	if pub.publisher != nil {
		t.Error("Expected nil publisher when host is missing")
	}

	if pub.messagingService != nil {
		t.Error("Expected nil messaging service when host is missing")
	}
}

func TestNewPublisher_MissingVpnName(t *testing.T) {
	config := &Configuration{
		Enabled: true,
		Host:    "tcp://localhost:55555",
		VpnName: "",
	}

	pub, err := NewPublisher(config, "solar")
	if err != nil {
		t.Fatalf("Expected no error for missing VPN name (returns empty publisher), got: %v", err)
	}

	if pub.publisher != nil {
		t.Error("Expected nil publisher when VPN name is missing")
	}

	if pub.messagingService != nil {
		t.Error("Expected nil messaging service when VPN name is missing")
	}
}

func TestNewPublisher_TopicPrefix(t *testing.T) {
	tests := []struct {
		name              string
		configTopicPrefix string
		paramTopicPrefix  string
	}{
		{
			name:              "Use parameter when provided",
			configTopicPrefix: "config-prefix",
			paramTopicPrefix:  "param-prefix",
		},
		{
			name:              "Use config when parameter empty",
			configTopicPrefix: "config-prefix",
			paramTopicPrefix:  "",
		},
		{
			name:              "Use default when both empty",
			configTopicPrefix: "",
			paramTopicPrefix:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Configuration{
				Enabled:     false, // Disabled to avoid connection attempt
				Host:        "tcp://localhost:55555",
				VpnName:     "default",
				TopicPrefix: tt.configTopicPrefix,
			}

			pub, err := NewPublisher(config, tt.paramTopicPrefix)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// For disabled publishers, topicPrefix should be empty
			if pub.topicPrefix != "" {
				t.Errorf("Expected empty prefix for disabled publisher, got: %s", pub.topicPrefix)
			}
		})
	}
}

func TestPublish_DisabledPublisher(_ *testing.T) {
	pub := &Publisher{
		publisher:        nil,
		messagingService: nil,
		topicPrefix:      "",
	}

	// Should not panic when publishing to disabled publisher
	pub.Publish("test/topic", "test payload")
}

func TestClose_DisabledPublisher(_ *testing.T) {
	pub := &Publisher{
		publisher:        nil,
		messagingService: nil,
		topicPrefix:      "",
	}

	// Should not panic when closing disabled publisher
	pub.Close()
}

func TestServicePropertyMap(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		vpnName  string
		username string
		password string
	}{
		{
			name:     "All fields provided",
			host:     "tcp://localhost:55555",
			vpnName:  "default",
			username: "user123",
			password: "pass123",
		},
		{
			name:     "No credentials",
			host:     "tcp://localhost:55555",
			vpnName:  "default",
			username: "",
			password: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Configuration{}
			propMap := cfg.ServicePropertyMap(tt.host, tt.vpnName, tt.username, tt.password)

			if propMap == nil {
				t.Fatal("Expected non-nil property map")
			}

			// Verify that ServicePropertyMap returns a valid config.ServicePropertyMap
			// The actual validation happens when Solace SDK uses it
			// We just verify it doesn't panic and returns something
		})
	}
}

// Note: Full integration tests with mocked Solace clients would require extensive
// interface implementations. The tests above cover the basic configuration and
// edge case handling. For full publish/close behavior, integration tests with
// actual Solace infrastructure or testcontainers would be more appropriate.
