package voltgo

import (
	"context"
	"fmt"
	"sync"

	"github.com/lumberbarons/voltgo/battery"
)

// MockBatteryClient is a mock implementation of the BatteryClient interface for testing.
type MockBatteryClient struct {
	mu sync.RWMutex

	// Function fields that can be set to customize behavior in tests
	GetStatusFunc   func(ctx context.Context) (*battery.Status, error)
	GetInfoFunc     func(ctx context.Context) (*battery.Info, error)
	IsConnectedFunc func() bool
	DisconnectFunc  func() error

	// Call tracking
	GetStatusCalls   int
	GetInfoCalls     int
	IsConnectedCalls int
	DisconnectCalls  int
}

// Verify MockBatteryClient implements BatteryClient
var _ BatteryClient = (*MockBatteryClient)(nil)

func (m *MockBatteryClient) GetStatus(ctx context.Context) (*battery.Status, error) {
	m.mu.Lock()
	m.GetStatusCalls++
	m.mu.Unlock()

	if m.GetStatusFunc != nil {
		return m.GetStatusFunc(ctx)
	}
	return nil, fmt.Errorf("GetStatus not implemented")
}

func (m *MockBatteryClient) GetInfo(ctx context.Context) (*battery.Info, error) {
	m.mu.Lock()
	m.GetInfoCalls++
	m.mu.Unlock()

	if m.GetInfoFunc != nil {
		return m.GetInfoFunc(ctx)
	}
	return nil, fmt.Errorf("GetInfo not implemented")
}

func (m *MockBatteryClient) IsConnected() bool {
	m.mu.Lock()
	m.IsConnectedCalls++
	m.mu.Unlock()

	if m.IsConnectedFunc != nil {
		return m.IsConnectedFunc()
	}
	return true
}

func (m *MockBatteryClient) Disconnect() error {
	m.mu.Lock()
	m.DisconnectCalls++
	m.mu.Unlock()

	if m.DisconnectFunc != nil {
		return m.DisconnectFunc()
	}
	return nil
}

// MockBatteryConnector is a mock implementation of the BatteryConnector interface for testing.
type MockBatteryConnector struct {
	mu sync.RWMutex

	// Function fields that can be set to customize behavior in tests
	ConnectFunc func(ctx context.Context, address string) (BatteryClient, error)
	CloseFunc   func() error

	// Call tracking
	ConnectCalls []string
	CloseCalls   int
}

// Verify MockBatteryConnector implements BatteryConnector
var _ BatteryConnector = (*MockBatteryConnector)(nil)

func (m *MockBatteryConnector) Connect(ctx context.Context, address string) (BatteryClient, error) {
	m.mu.Lock()
	m.ConnectCalls = append(m.ConnectCalls, address)
	m.mu.Unlock()

	if m.ConnectFunc != nil {
		return m.ConnectFunc(ctx, address)
	}
	return nil, fmt.Errorf("Connect not implemented")
}

func (m *MockBatteryConnector) Close() error {
	m.mu.Lock()
	m.CloseCalls++
	m.mu.Unlock()

	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}
