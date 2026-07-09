package solace

import (
	"io"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"solace.dev/go/messaging/pkg/solace/config"
)

func TestMain(m *testing.M) {
	// Suppress log output during tests without mutating the global level
	origOutput := logrus.StandardLogger().Out
	logrus.SetOutput(io.Discard)
	code := m.Run()
	logrus.SetOutput(origOutput)
	os.Exit(code)
}

func TestNewPublisher_Disabled(t *testing.T) {
	config := &Configuration{
		Enabled: false,
		Host:    "tcp://localhost:55555",
		VpnName: "default",
	}

	pub, err := NewPublisher(config)
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

	pub, err := NewPublisher(config)
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

	pub, err := NewPublisher(config)
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
		expectedPrefix string
	}{
		{
			name:           "Use config when provided",
			configPrefix:   "config-prefix",
			expectedPrefix: "config-prefix",
		},
		{
			name:           "Use default when config empty",
			configPrefix:   "",
			expectedPrefix: "solar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveTopicPrefix(tt.configPrefix)
			if result != tt.expectedPrefix {
				t.Errorf("resolveTopicPrefix(%q) = %q, want %q",
					tt.configPrefix, result, tt.expectedPrefix)
			}
		})
	}
}

func TestPublish_DisabledPublisher(t *testing.T) {
	pub := &Publisher{
		publisher:        nil,
		messagingService: nil,
		topicPrefix:      "",
	}

	// Should not panic and fields should remain nil (no side effects)
	pub.Publish("test/topic", "test payload")

	if pub.publisher != nil {
		t.Error("Expected publisher to remain nil after publishing to disabled publisher")
	}
}

func TestClose_DisabledPublisher(t *testing.T) {
	pub := &Publisher{
		publisher:        nil,
		messagingService: nil,
		topicPrefix:      "",
	}

	// Should not panic and fields should remain nil (no side effects)
	pub.Close()

	if pub.publisher != nil {
		t.Error("Expected publisher to remain nil after closing disabled publisher")
	}
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

func TestCredentialsOverPlaintext(t *testing.T) {
	tests := []struct {
		name   string
		config Configuration
		want   bool
	}{
		{
			name:   "credentials with plaintext tcp scheme",
			config: Configuration{Host: "tcp://broker:55555", Username: "user", Password: "pass"},
			want:   true,
		},
		{
			name:   "credentials with tcps scheme",
			config: Configuration{Host: "tcps://broker:55443", Username: "user", Password: "pass"},
			want:   false,
		},
		{
			name:   "no credentials with plaintext scheme",
			config: Configuration{Host: "tcp://broker:55555"},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := credentialsOverPlaintext(&tt.config); got != tt.want {
				t.Errorf("credentialsOverPlaintext() = %v, want %v", got, tt.want)
			}
		})
	}
}
