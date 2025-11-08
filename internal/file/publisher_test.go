package file

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewPublisher_Disabled(t *testing.T) {
	config := &Configuration{
		Enabled: false,
	}

	publisher, err := NewPublisher(config, "test/prefix")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if publisher.logger != nil {
		t.Error("expected nil logger for disabled publisher")
	}
}

func TestNewPublisher_NoFilename(t *testing.T) {
	config := &Configuration{
		Enabled:  true,
		Filename: "",
	}

	publisher, err := NewPublisher(config, "test/prefix")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if publisher.logger != nil {
		t.Error("expected nil logger when filename is not provided")
	}
}

func TestNewPublisher_WithDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.log")

	config := &Configuration{
		Enabled:  true,
		Filename: filename,
	}

	publisher, err := NewPublisher(config, "test/prefix")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer publisher.Close()

	if publisher.logger == nil {
		t.Fatal("expected logger to be initialized")
	}

	if publisher.logger.MaxSize != 10 {
		t.Errorf("expected default MaxSize of 10, got %d", publisher.logger.MaxSize)
	}

	if publisher.logger.MaxBackups != 10 {
		t.Errorf("expected default MaxBackups of 10, got %d", publisher.logger.MaxBackups)
	}
}

func TestNewPublisher_WithCustomConfig(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.log")

	config := &Configuration{
		Enabled:    true,
		Filename:   filename,
		MaxSizeMB:  5,
		MaxBackups: 3,
		Compress:   true,
	}

	publisher, err := NewPublisher(config, "test/prefix")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer publisher.Close()

	if publisher.logger.MaxSize != 5 {
		t.Errorf("expected MaxSize of 5, got %d", publisher.logger.MaxSize)
	}

	if publisher.logger.MaxBackups != 3 {
		t.Errorf("expected MaxBackups of 3, got %d", publisher.logger.MaxBackups)
	}

	if !publisher.logger.Compress {
		t.Error("expected Compress to be true")
	}
}

func TestPublish(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.log")

	config := &Configuration{
		Enabled:  true,
		Filename: filename,
	}

	publisher, err := NewPublisher(config, "test/prefix")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer publisher.Close()

	// Publish a test message
	topicSuffix := "epever/battery-voltage"
	payload := `{"value":12.5,"unit":"volts","timestamp":1699000000}`

	publisher.Publish(topicSuffix, payload)

	// Read the file and verify content
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	content := string(data)
	expectedTopic := "test/prefix/epever/battery-voltage"
	if !strings.Contains(content, expectedTopic) {
		t.Errorf("expected log to contain topic %q, got %q", expectedTopic, content)
	}

	// Verify it's valid JSON
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	var logEntry struct {
		Timestamp string          `json:"timestamp"`
		Topic     string          `json:"topic"`
		Payload   json.RawMessage `json:"payload"`
	}

	if err := json.Unmarshal([]byte(lines[0]), &logEntry); err != nil {
		t.Fatalf("failed to parse log entry as JSON: %v", err)
	}

	if logEntry.Topic != expectedTopic {
		t.Errorf("expected topic %q, got %q", expectedTopic, logEntry.Topic)
	}

	if logEntry.Timestamp == "" {
		t.Error("expected timestamp to be set")
	}

	// Verify the payload is the original JSON
	var payloadData map[string]interface{}
	if err := json.Unmarshal(logEntry.Payload, &payloadData); err != nil {
		t.Fatalf("failed to parse payload: %v", err)
	}

	if payloadData["value"].(float64) != 12.5 {
		t.Errorf("expected value 12.5, got %v", payloadData["value"])
	}
}

func TestPublish_MultipleMessages(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.log")

	config := &Configuration{
		Enabled:  true,
		Filename: filename,
	}

	publisher, err := NewPublisher(config, "test/prefix")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer publisher.Close()

	// Publish multiple messages
	messages := []struct {
		topic   string
		payload string
	}{
		{"epever/battery-voltage", `{"value":12.5,"unit":"volts"}`},
		{"epever/battery-current", `{"value":5.2,"unit":"amperes"}`},
		{"epever/battery-soc", `{"value":85,"unit":"percent"}`},
	}

	for _, msg := range messages {
		publisher.Publish(msg.topic, msg.payload)
	}

	// Read the file and verify all messages are present
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != len(messages) {
		t.Fatalf("expected %d lines, got %d", len(messages), len(lines))
	}

	for i, line := range lines {
		var logEntry struct {
			Topic string `json:"topic"`
		}
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			t.Fatalf("failed to parse line %d: %v", i, err)
		}

		expectedTopic := "test/prefix/" + messages[i].topic
		if logEntry.Topic != expectedTopic {
			t.Errorf("line %d: expected topic %q, got %q", i, expectedTopic, logEntry.Topic)
		}
	}
}

func TestPublish_DisabledPublisher(t *testing.T) {
	config := &Configuration{
		Enabled: false,
	}

	publisher, err := NewPublisher(config, "test/prefix")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should not panic when publishing to disabled publisher
	publisher.Publish("test/topic", `{"value":123}`)
}
