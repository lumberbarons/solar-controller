package publishers

import (
	"github.com/lumberbarons/solar-controller/internal/controllers"
	"github.com/sirupsen/logrus"
)

// MultiPublisher implements MessagePublisher by fanning out to multiple publishers.
// It provides best-effort delivery: if one publisher fails, others continue to receive messages.
type MultiPublisher struct {
	publishers []controllers.MessagePublisher
	logger     *logrus.Logger
}

// NewMultiPublisher creates a new MultiPublisher that fans out to all provided publishers.
func NewMultiPublisher(publishers []controllers.MessagePublisher, logger *logrus.Logger) *MultiPublisher {
	return &MultiPublisher{
		publishers: publishers,
		logger:     logger,
	}
}

// Publish sends the message to all configured publishers.
// Errors from individual publishers are logged but do not stop other publishers from receiving the message.
func (m *MultiPublisher) Publish(topicSuffix, payload string) {
	for _, publisher := range m.publishers {
		// Each publisher handles its own error logging internally
		// We just fan out the message
		publisher.Publish(topicSuffix, payload)
	}
}

// Close closes all configured publishers.
func (m *MultiPublisher) Close() {
	for _, publisher := range m.publishers {
		publisher.Close()
	}
}
