package sns

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
		Enabled:  false,
		Region:   "us-east-1",
		TopicArn: "arn:aws:sns:us-east-1:123456789012:test-topic",
	}

	pub, err := NewPublisher(config, "solar")
	if err != nil {
		t.Fatalf("Expected no error for disabled publisher, got: %v", err)
	}

	if pub.client != nil {
		t.Error("Expected nil client for disabled publisher")
	}

	if pub.topicPrefix != "" {
		t.Errorf("Expected empty topic prefix for disabled publisher, got: %s", pub.topicPrefix)
	}
}

func TestNewPublisher_MissingTopicArn(t *testing.T) {
	config := &Configuration{
		Enabled:  true,
		Region:   "us-east-1",
		TopicArn: "",
	}

	pub, err := NewPublisher(config, "solar")
	if err != nil {
		t.Fatalf("Expected no error for missing topic ARN (returns empty publisher), got: %v", err)
	}

	if pub.client != nil {
		t.Error("Expected nil client when topic ARN is missing")
	}

	if pub.topicPrefix != "" {
		t.Errorf("Expected empty topic prefix when topic ARN is missing, got: %s", pub.topicPrefix)
	}
}

func TestNewPublisher_MissingRegion(t *testing.T) {
	config := &Configuration{
		Enabled:  true,
		Region:   "",
		TopicArn: "arn:aws:sns:us-east-1:123456789012:test-topic",
	}

	pub, err := NewPublisher(config, "solar")
	if err != nil {
		t.Fatalf("Expected no error for missing region (returns empty publisher), got: %v", err)
	}

	if pub.client != nil {
		t.Error("Expected nil client when region is missing")
	}

	if pub.topicPrefix != "" {
		t.Errorf("Expected empty topic prefix when region is missing, got: %s", pub.topicPrefix)
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
		client:      nil,
		topicPrefix: "",
	}

	// Should not panic when publishing to disabled publisher
	pub.Publish("test/topic", "test payload")
}

func TestClose_DisabledPublisher(_ *testing.T) {
	pub := &Publisher{
		client:      nil,
		topicPrefix: "",
	}

	// Should not panic when closing disabled publisher
	pub.Close()
}

// Note: Full integration tests with mocked SNS clients would require extensive
// interface implementations. The tests above cover the basic configuration and
// edge case handling. For full publish behavior, integration tests with
// actual AWS SNS infrastructure or LocalStack would be more appropriate.
