package testutil

import (
	"sync"

	"github.com/lumberbarons/solar-controller/internal/publish"
)

// MockMessagePublisher is a mock implementation of the MessagePublisher interface for testing.
type MockMessagePublisher struct {
	mu sync.RWMutex

	// Function fields that can be set to customize behavior in tests
	PublishFunc func(topicSuffix, payload string)
	CloseFunc   func()

	// Call tracking
	PublishCalls []PublishCall
	CloseCalls   int
}

type PublishCall struct {
	TopicSuffix string
	Payload     string
}

// Verify MockMessagePublisher implements MessagePublisher
var _ publish.MessagePublisher = (*MockMessagePublisher)(nil)

func (m *MockMessagePublisher) Publish(topicSuffix, payload string) {
	m.mu.Lock()
	m.PublishCalls = append(m.PublishCalls, PublishCall{TopicSuffix: topicSuffix, Payload: payload})
	m.mu.Unlock()

	if m.PublishFunc != nil {
		m.PublishFunc(topicSuffix, payload)
	}
}

func (m *MockMessagePublisher) Close() {
	m.mu.Lock()
	m.CloseCalls++
	m.mu.Unlock()

	if m.CloseFunc != nil {
		m.CloseFunc()
	}
}

// NewMockPublisher creates a new MockMessagePublisher with a Closed tracking field.
func NewMockPublisher() *MockPublisher {
	return &MockPublisher{}
}

// MockPublisher extends MockMessagePublisher with a Closed field for testing.
type MockPublisher struct {
	MockMessagePublisher
	Closed bool
}

func (m *MockPublisher) Close() {
	m.Closed = true
	m.MockMessagePublisher.Close()
}
