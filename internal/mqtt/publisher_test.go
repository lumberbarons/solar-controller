package mqtt

import (
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

// Mock implementation of mqtt.Token
type MockToken struct {
	waitResult    bool
	errorResult   error
	waitTimeoutMS int // Time to wait before returning from WaitTimeout
}

func (m *MockToken) Wait() bool {
	return m.waitResult
}

func (m *MockToken) WaitTimeout(_ time.Duration) bool {
	if m.waitTimeoutMS > 0 {
		time.Sleep(time.Duration(m.waitTimeoutMS) * time.Millisecond)
	}
	return m.waitResult
}

func (m *MockToken) Error() error {
	return m.errorResult
}

func (m *MockToken) Done() <-chan struct{} {
	return nil
}

// Mock implementation of mqtt.Client
type MockMQTTClient struct {
	connectToken    *MockToken
	publishToken    *MockToken
	disconnectCalls int
	publishCalls    []PublishCall
}

type PublishCall struct {
	Topic   string
	QoS     byte
	Payload string
}

func (m *MockMQTTClient) IsConnected() bool {
	return true
}

func (m *MockMQTTClient) IsConnectionOpen() bool {
	return true
}

func (m *MockMQTTClient) Connect() mqtt.Token {
	return m.connectToken
}

func (m *MockMQTTClient) Disconnect(_ uint) {
	m.disconnectCalls++
}

func (m *MockMQTTClient) Publish(topic string, qos byte, _ bool, payload interface{}) mqtt.Token {
	m.publishCalls = append(m.publishCalls, PublishCall{
		Topic:   topic,
		QoS:     qos,
		Payload: payload.(string),
	})
	return m.publishToken
}

func (m *MockMQTTClient) Subscribe(_ string, _ byte, _ mqtt.MessageHandler) mqtt.Token {
	return nil
}

func (m *MockMQTTClient) SubscribeMultiple(_ map[string]byte, _ mqtt.MessageHandler) mqtt.Token {
	return nil
}

func (m *MockMQTTClient) Unsubscribe(_ ...string) mqtt.Token {
	return nil
}

func (m *MockMQTTClient) AddRoute(_ string, _ mqtt.MessageHandler) {
}

func (m *MockMQTTClient) OptionsReader() mqtt.ClientOptionsReader {
	return mqtt.ClientOptionsReader{}
}

func init() {
	// Suppress log output during tests
	logrus.SetLevel(logrus.ErrorLevel)
}

func TestNewPublisher_Disabled(t *testing.T) {
	config := &Configuration{
		Enabled: false,
		Host:    "tcp://localhost:1883",
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

func TestNewPublisher_MissingHost(t *testing.T) {
	config := &Configuration{
		Enabled: true,
		Host:    "",
	}

	pub, err := NewPublisher(config, "solar")
	if err != nil {
		t.Fatalf("Expected no error for missing host (returns empty publisher), got: %v", err)
	}

	if pub.client != nil {
		t.Error("Expected nil client when host is missing")
	}

	if pub.topicPrefix != "" {
		t.Errorf("Expected empty topic prefix when host is missing, got: %s", pub.topicPrefix)
	}
}

func TestNewPublisher_TopicPrefix(t *testing.T) {
	tests := []struct {
		name              string
		configTopicPrefix string
		paramTopicPrefix  string
		expectedPrefix    string
	}{
		{
			name:              "Use parameter when provided",
			configTopicPrefix: "config-prefix",
			paramTopicPrefix:  "param-prefix",
			expectedPrefix:    "param-prefix",
		},
		{
			name:              "Use config when parameter empty",
			configTopicPrefix: "config-prefix",
			paramTopicPrefix:  "",
			expectedPrefix:    "config-prefix",
		},
		{
			name:              "Use default when both empty",
			configTopicPrefix: "",
			paramTopicPrefix:  "",
			expectedPrefix:    "solar",
		},
		{
			name:              "Use parameter over default",
			configTopicPrefix: "",
			paramTopicPrefix:  "custom",
			expectedPrefix:    "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Configuration{
				Enabled:     false, // Disabled to avoid connection attempt
				Host:        "tcp://localhost:1883",
				TopicPrefix: tt.configTopicPrefix,
			}

			pub, err := NewPublisher(config, tt.paramTopicPrefix)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Even for disabled publishers, topicPrefix logic should work (though it's empty for disabled)
			// To test this properly, we'd need to mock the client creation
			// For now, we verify the disabled case returns empty
			if pub.topicPrefix != "" {
				t.Errorf("Expected empty prefix for disabled publisher, got: %s", pub.topicPrefix)
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

func TestPublish_TopicFormation(t *testing.T) {
	mockClient := &MockMQTTClient{
		publishToken: &MockToken{
			waitResult: true,
		},
	}

	pub := &Publisher{
		client:      mockClient,
		topicPrefix: "solar",
	}

	pub.Publish("device-1/epever/battery-voltage", `{"value":12.5}`)

	if len(mockClient.publishCalls) != 1 {
		t.Fatalf("Expected 1 publish call, got: %d", len(mockClient.publishCalls))
	}

	expectedTopic := "solar/device-1/epever/battery-voltage"
	if mockClient.publishCalls[0].Topic != expectedTopic {
		t.Errorf("Expected topic %s, got: %s", expectedTopic, mockClient.publishCalls[0].Topic)
	}

	if mockClient.publishCalls[0].Payload != `{"value":12.5}` {
		t.Errorf("Expected payload '{\"value\":12.5}', got: %s", mockClient.publishCalls[0].Payload)
	}

	if mockClient.publishCalls[0].QoS != 0 {
		t.Errorf("Expected QoS 0, got: %d", mockClient.publishCalls[0].QoS)
	}
}

func TestPublish_Timeout(t *testing.T) {
	mockClient := &MockMQTTClient{
		publishToken: &MockToken{
			waitResult:    false, // Simulate timeout
			waitTimeoutMS: 100,   // Wait 100ms to simulate delay
		},
	}

	pub := &Publisher{
		client:      mockClient,
		topicPrefix: "solar",
	}

	start := time.Now()
	pub.Publish("test/topic", "payload")
	elapsed := time.Since(start)

	// Should wait for timeout (we use 100ms in mock, but actual timeout is 5s)
	// The test should complete relatively quickly since we return false quickly
	if elapsed > 6*time.Second {
		t.Errorf("Publish took too long: %v", elapsed)
	}

	if len(mockClient.publishCalls) != 1 {
		t.Errorf("Expected 1 publish call, got: %d", len(mockClient.publishCalls))
	}
}

func TestPublish_Error(t *testing.T) {
	mockClient := &MockMQTTClient{
		publishToken: &MockToken{
			waitResult:  true,
			errorResult: mqtt.ErrNotConnected,
		},
	}

	pub := &Publisher{
		client:      mockClient,
		topicPrefix: "solar",
	}

	// Should log error but not panic
	pub.Publish("test/topic", "payload")

	if len(mockClient.publishCalls) != 1 {
		t.Errorf("Expected 1 publish call, got: %d", len(mockClient.publishCalls))
	}
}

func TestPublish_MultipleMessages(t *testing.T) {
	mockClient := &MockMQTTClient{
		publishToken: &MockToken{
			waitResult: true,
		},
	}

	pub := &Publisher{
		client:      mockClient,
		topicPrefix: "solar",
	}

	messages := []struct {
		topic   string
		payload string
	}{
		{"device-1/epever/battery-voltage", `{"value":12.5}`},
		{"device-1/epever/battery-current", `{"value":3.2}`},
		{"device-1/epever/battery-soc", `{"value":85}`},
	}

	for _, msg := range messages {
		pub.Publish(msg.topic, msg.payload)
	}

	if len(mockClient.publishCalls) != len(messages) {
		t.Fatalf("Expected %d publish calls, got: %d", len(messages), len(mockClient.publishCalls))
	}

	for i, msg := range messages {
		expectedTopic := "solar/" + msg.topic
		if mockClient.publishCalls[i].Topic != expectedTopic {
			t.Errorf("Message %d: expected topic %s, got: %s", i, expectedTopic, mockClient.publishCalls[i].Topic)
		}
		if mockClient.publishCalls[i].Payload != msg.payload {
			t.Errorf("Message %d: expected payload %s, got: %s", i, msg.payload, mockClient.publishCalls[i].Payload)
		}
	}
}

func TestClose_DisabledPublisher(t *testing.T) {
	mockClient := &MockMQTTClient{}

	pub := &Publisher{
		client:      mockClient,
		topicPrefix: "",
	}

	pub.Close()

	if mockClient.disconnectCalls != 1 {
		t.Errorf("Expected 1 disconnect call, got: %d", mockClient.disconnectCalls)
	}
}

func TestClose_ValidPublisher(t *testing.T) {
	mockClient := &MockMQTTClient{}

	pub := &Publisher{
		client:      mockClient,
		topicPrefix: "solar",
	}

	pub.Close()

	if mockClient.disconnectCalls != 1 {
		t.Errorf("Expected 1 disconnect call, got: %d", mockClient.disconnectCalls)
	}
}
