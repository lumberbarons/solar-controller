package solace

import (
	"testing"

	"github.com/sirupsen/logrus"
	"solace.dev/go/messaging/pkg/solace/config"
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

func TestResolveTopicPrefix(t *testing.T) {
	tests := []struct {
		name           string
		configPrefix   string
		paramPrefix    string
		expectedPrefix string
	}{
		{
			name:           "Use parameter when provided",
			configPrefix:   "config-prefix",
			paramPrefix:    "param-prefix",
			expectedPrefix: "param-prefix",
		},
		{
			name:           "Use config when parameter empty",
			configPrefix:   "config-prefix",
			paramPrefix:    "",
			expectedPrefix: "config-prefix",
		},
		{
			name:           "Use default when both empty",
			configPrefix:   "",
			paramPrefix:    "",
			expectedPrefix: "solar",
		},
		{
			name:           "Use parameter over default",
			configPrefix:   "",
			paramPrefix:    "custom",
			expectedPrefix: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveTopicPrefix(tt.configPrefix, tt.paramPrefix)
			if result != tt.expectedPrefix {
				t.Errorf("resolveTopicPrefix(%q, %q) = %q, want %q",
					tt.configPrefix, tt.paramPrefix, result, tt.expectedPrefix)
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

			if got := propMap[config.TransportLayerPropertyHost]; got != tt.host {
				t.Errorf("Host = %v, want %v", got, tt.host)
			}
			if got := propMap[config.ServicePropertyVPNName]; got != tt.vpnName {
				t.Errorf("VPNName = %v, want %v", got, tt.vpnName)
			}
			if got := propMap[config.AuthenticationPropertySchemeBasicUserName]; got != tt.username {
				t.Errorf("Username = %v, want %v", got, tt.username)
			}
			if got := propMap[config.AuthenticationPropertySchemeBasicPassword]; got != tt.password {
				t.Errorf("Password = %v, want %v", got, tt.password)
			}
		})
	}
}

// Note: Full integration tests with mocked Solace clients would require extensive
// interface implementations. The tests above cover the basic configuration and
// edge case handling. For full publish/close behavior, integration tests with
// actual Solace infrastructure or testcontainers would be more appropriate.
