package containers

import (
	"context"
	"fmt"
	"testing"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
)

// LocalStackContainer wraps a LocalStack testcontainer with helper methods
type LocalStackContainer struct {
	Container *localstack.LocalStackContainer
}

// StartLocalStack creates and starts a LocalStack container with SNS enabled
func StartLocalStack(t *testing.T) *LocalStackContainer {
	t.Helper()

	ctx := context.Background()

	// Start LocalStack container
	container, err := localstack.Run(ctx,
		"localstack/localstack:3.0",
	)
	if err != nil {
		t.Fatalf("failed to start LocalStack container: %v", err)
	}

	// Ensure cleanup on test completion
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Logf("failed to cleanup LocalStack container: %v", err)
		}
	})

	return &LocalStackContainer{
		Container: container,
	}
}

// GetSNSEndpoint returns the SNS endpoint URL
func (l *LocalStackContainer) GetSNSEndpoint(t *testing.T) string {
	t.Helper()

	ctx := context.Background()

	// Get mapped port for LocalStack (uses 4566 internally)
	mappedPort, err := l.Container.MappedPort(ctx, nat.Port("4566/tcp"))
	if err != nil {
		t.Fatalf("failed to get mapped port: %v", err)
	}

	// Get Docker host
	provider, err := testcontainers.NewDockerProvider()
	if err != nil {
		t.Fatalf("failed to get Docker provider: %v", err)
	}
	defer provider.Close()

	host, err := provider.DaemonHost(ctx)
	if err != nil {
		t.Fatalf("failed to get daemon host: %v", err)
	}

	// Construct endpoint URL
	endpoint := fmt.Sprintf("http://%s:%s", host, mappedPort.Port())
	return endpoint
}

// GetRegion returns the AWS region for LocalStack (always us-east-1)
func (l *LocalStackContainer) GetRegion() string {
	return "us-east-1"
}

// GetCredentials returns dummy AWS credentials for LocalStack
func (l *LocalStackContainer) GetCredentials() (accessKey, secretKey string) {
	// LocalStack accepts any credentials
	return "test", "test"
}
