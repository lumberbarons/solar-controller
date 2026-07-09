// Package publish defines the MessagePublisher interface shared between
// controllers (which publish metrics) and publisher implementations.
package publish

// MessagePublisher defines the interface for publishing messages to a message broker.
// This abstraction allows for testing without a real message broker.
type MessagePublisher interface {
	// Publish publishes a message with the given topic suffix and payload.
	Publish(topicSuffix, payload string)

	// Close closes the publisher connection.
	Close()
}

// NoOpPublisher is a publisher that does nothing.
// Used when no message publisher is configured.
type NoOpPublisher struct{}

func (n *NoOpPublisher) Publish(_, _ string) {
	// Do nothing
}

func (n *NoOpPublisher) Close() {
	// Do nothing
}
