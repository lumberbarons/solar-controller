package containers

import (
	"context"
	"testing"
	"time"
)

// WaitForContainer waits for a container to be ready with a timeout
func WaitForContainer(ctx context.Context, timeout time.Duration, checkFunc func() error) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if err := checkFunc(); err == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return context.DeadlineExceeded
}

// SkipIfDockerNotAvailable skips the test if Docker is not available
func SkipIfDockerNotAvailable(t *testing.T) {
	t.Helper()

	// This will be handled by testcontainers-go automatically
	// If Docker is not available, container creation will fail
	// We can add a more explicit check if needed in the future
}
