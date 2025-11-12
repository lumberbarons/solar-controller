package publishers

import (
	"testing"

	"github.com/lumberbarons/solar-controller/internal/controllers"
	"github.com/sirupsen/logrus"
)

// MockPublisher is a test double for MessagePublisher
type MockPublisher struct {
	publishCalls []publishCall
	closeCalled  bool
}

type publishCall struct {
	topicSuffix string
	payload     string
}

func (m *MockPublisher) Publish(topicSuffix, payload string) {
	m.publishCalls = append(m.publishCalls, publishCall{
		topicSuffix: topicSuffix,
		payload:     payload,
	})
}

func (m *MockPublisher) Close() {
	m.closeCalled = true
}

// Ensure MockPublisher implements MessagePublisher
var _ controllers.MessagePublisher = (*MockPublisher)(nil)

func TestMultiPublisher_Publish_FansOutToAllPublishers(t *testing.T) {
	// Arrange
	mock1 := &MockPublisher{}
	mock2 := &MockPublisher{}
	mock3 := &MockPublisher{}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Suppress logs during test

	multiPublisher := NewMultiPublisher([]controllers.MessagePublisher{mock1, mock2, mock3}, logger)

	// Act
	multiPublisher.Publish("device-1/epever/battery-voltage", `{"value":12.5,"unit":"volts","timestamp":1234567890}`)

	// Assert
	if len(mock1.publishCalls) != 1 {
		t.Errorf("Expected 1 publish call to mock1, got %d", len(mock1.publishCalls))
	}
	if len(mock2.publishCalls) != 1 {
		t.Errorf("Expected 1 publish call to mock2, got %d", len(mock2.publishCalls))
	}
	if len(mock3.publishCalls) != 1 {
		t.Errorf("Expected 1 publish call to mock3, got %d", len(mock3.publishCalls))
	}

	// Verify all received the same message
	expectedTopic := "device-1/epever/battery-voltage"
	expectedPayload := `{"value":12.5,"unit":"volts","timestamp":1234567890}`

	if mock1.publishCalls[0].topicSuffix != expectedTopic {
		t.Errorf("Mock1 received wrong topic: %s", mock1.publishCalls[0].topicSuffix)
	}
	if mock1.publishCalls[0].payload != expectedPayload {
		t.Errorf("Mock1 received wrong payload: %s", mock1.publishCalls[0].payload)
	}

	if mock2.publishCalls[0].topicSuffix != expectedTopic {
		t.Errorf("Mock2 received wrong topic: %s", mock2.publishCalls[0].topicSuffix)
	}
	if mock2.publishCalls[0].payload != expectedPayload {
		t.Errorf("Mock2 received wrong payload: %s", mock2.publishCalls[0].payload)
	}

	if mock3.publishCalls[0].topicSuffix != expectedTopic {
		t.Errorf("Mock3 received wrong topic: %s", mock3.publishCalls[0].topicSuffix)
	}
	if mock3.publishCalls[0].payload != expectedPayload {
		t.Errorf("Mock3 received wrong payload: %s", mock3.publishCalls[0].payload)
	}
}

func TestMultiPublisher_PublishMultipleTimes(t *testing.T) {
	// Arrange
	mock1 := &MockPublisher{}
	mock2 := &MockPublisher{}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	multiPublisher := NewMultiPublisher([]controllers.MessagePublisher{mock1, mock2}, logger)

	// Act - publish 3 different metrics
	multiPublisher.Publish("device-1/epever/battery-voltage", `{"value":12.5}`)
	multiPublisher.Publish("device-1/epever/battery-current", `{"value":5.2}`)
	multiPublisher.Publish("device-1/epever/battery-soc", `{"value":85}`)

	// Assert
	if len(mock1.publishCalls) != 3 {
		t.Errorf("Expected 3 publish calls to mock1, got %d", len(mock1.publishCalls))
	}
	if len(mock2.publishCalls) != 3 {
		t.Errorf("Expected 3 publish calls to mock2, got %d", len(mock2.publishCalls))
	}

	// Verify order is preserved
	if mock1.publishCalls[0].topicSuffix != "device-1/epever/battery-voltage" {
		t.Errorf("Wrong order for mock1 call 0")
	}
	if mock1.publishCalls[1].topicSuffix != "device-1/epever/battery-current" {
		t.Errorf("Wrong order for mock1 call 1")
	}
	if mock1.publishCalls[2].topicSuffix != "device-1/epever/battery-soc" {
		t.Errorf("Wrong order for mock1 call 2")
	}
}

func TestMultiPublisher_Close_ClosesAllPublishers(t *testing.T) {
	// Arrange
	mock1 := &MockPublisher{}
	mock2 := &MockPublisher{}
	mock3 := &MockPublisher{}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	multiPublisher := NewMultiPublisher([]controllers.MessagePublisher{mock1, mock2, mock3}, logger)

	// Act
	multiPublisher.Close()

	// Assert
	if !mock1.closeCalled {
		t.Error("Expected mock1.Close() to be called")
	}
	if !mock2.closeCalled {
		t.Error("Expected mock2.Close() to be called")
	}
	if !mock3.closeCalled {
		t.Error("Expected mock3.Close() to be called")
	}
}

func TestMultiPublisher_EmptyPublishers(_ *testing.T) {
	// Arrange
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	multiPublisher := NewMultiPublisher([]controllers.MessagePublisher{}, logger)

	// Act & Assert - should not panic
	multiPublisher.Publish("device-1/epever/test", `{"value":1}`)
	multiPublisher.Close()
}

func TestMultiPublisher_SinglePublisher(t *testing.T) {
	// Arrange
	mock := &MockPublisher{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	multiPublisher := NewMultiPublisher([]controllers.MessagePublisher{mock}, logger)

	// Act
	multiPublisher.Publish("device-1/epever/test", `{"value":1}`)
	multiPublisher.Close()

	// Assert
	if len(mock.publishCalls) != 1 {
		t.Errorf("Expected 1 publish call, got %d", len(mock.publishCalls))
	}
	if !mock.closeCalled {
		t.Error("Expected Close() to be called")
	}
}
