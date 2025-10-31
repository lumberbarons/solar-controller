package testing

import (
	"context"
	"fmt"
	"sync"

	"github.com/lumberbarons/solar-controller/internal/controllers"
)

// MockModbusClient is a mock implementation of the ModbusClient interface for testing.
type MockModbusClient struct {
	mu sync.RWMutex

	// Function fields that can be set to customize behavior in tests
	ReadInputRegistersFunc     func(ctx context.Context, address, quantity uint16) ([]byte, error)
	ReadHoldingRegistersFunc   func(ctx context.Context, address, quantity uint16) ([]byte, error)
	WriteSingleRegisterFunc    func(ctx context.Context, address, value uint16) ([]byte, error)
	WriteMultipleRegistersFunc func(ctx context.Context, address, quantity uint16, value []byte) ([]byte, error)
	ReadCoilsFunc              func(ctx context.Context, address, quantity uint16) ([]byte, error)
	WriteSingleCoilFunc        func(ctx context.Context, address, value uint16) ([]byte, error)
	CloseFunc                  func()

	// Call tracking
	ReadInputRegistersCalls     []ReadRegistersCall
	ReadHoldingRegistersCalls   []ReadRegistersCall
	WriteSingleRegisterCalls    []WriteSingleRegisterCall
	WriteMultipleRegistersCalls []WriteMultipleRegistersCall
	ReadCoilsCalls              []ReadRegistersCall
	WriteSingleCoilCalls        []WriteSingleRegisterCall
	CloseCalls                  int
}

type ReadRegistersCall struct {
	Address  uint16
	Quantity uint16
}

type WriteSingleRegisterCall struct {
	Address uint16
	Value   uint16
}

type WriteMultipleRegistersCall struct {
	Address  uint16
	Quantity uint16
	Value    []byte
}

// Verify MockModbusClient implements ModbusClient
var _ controllers.ModbusClient = (*MockModbusClient)(nil)

func (m *MockModbusClient) ReadInputRegisters(ctx context.Context, address, quantity uint16) ([]byte, error) {
	m.mu.Lock()
	m.ReadInputRegistersCalls = append(m.ReadInputRegistersCalls, ReadRegistersCall{Address: address, Quantity: quantity})
	m.mu.Unlock()

	if m.ReadInputRegistersFunc != nil {
		return m.ReadInputRegistersFunc(ctx, address, quantity)
	}
	return nil, fmt.Errorf("ReadInputRegisters not implemented")
}

func (m *MockModbusClient) ReadHoldingRegisters(ctx context.Context, address, quantity uint16) ([]byte, error) {
	m.mu.Lock()
	m.ReadHoldingRegistersCalls = append(m.ReadHoldingRegistersCalls, ReadRegistersCall{Address: address, Quantity: quantity})
	m.mu.Unlock()

	if m.ReadHoldingRegistersFunc != nil {
		return m.ReadHoldingRegistersFunc(ctx, address, quantity)
	}
	return nil, fmt.Errorf("ReadHoldingRegisters not implemented")
}

func (m *MockModbusClient) WriteSingleRegister(ctx context.Context, address, value uint16) ([]byte, error) {
	m.mu.Lock()
	m.WriteSingleRegisterCalls = append(m.WriteSingleRegisterCalls, WriteSingleRegisterCall{Address: address, Value: value})
	m.mu.Unlock()

	if m.WriteSingleRegisterFunc != nil {
		return m.WriteSingleRegisterFunc(ctx, address, value)
	}
	return nil, nil
}

func (m *MockModbusClient) WriteMultipleRegisters(ctx context.Context, address, quantity uint16, value []byte) ([]byte, error) {
	m.mu.Lock()
	m.WriteMultipleRegistersCalls = append(m.WriteMultipleRegistersCalls, WriteMultipleRegistersCall{
		Address:  address,
		Quantity: quantity,
		Value:    value,
	})
	m.mu.Unlock()

	if m.WriteMultipleRegistersFunc != nil {
		return m.WriteMultipleRegistersFunc(ctx, address, quantity, value)
	}
	return nil, nil
}

func (m *MockModbusClient) ReadCoils(ctx context.Context, address, quantity uint16) ([]byte, error) {
	m.mu.Lock()
	m.ReadCoilsCalls = append(m.ReadCoilsCalls, ReadRegistersCall{Address: address, Quantity: quantity})
	m.mu.Unlock()

	if m.ReadCoilsFunc != nil {
		return m.ReadCoilsFunc(ctx, address, quantity)
	}
	return nil, fmt.Errorf("ReadCoils not implemented")
}

func (m *MockModbusClient) WriteSingleCoil(ctx context.Context, address, value uint16) ([]byte, error) {
	m.mu.Lock()
	m.WriteSingleCoilCalls = append(m.WriteSingleCoilCalls, WriteSingleRegisterCall{Address: address, Value: value})
	m.mu.Unlock()

	if m.WriteSingleCoilFunc != nil {
		return m.WriteSingleCoilFunc(ctx, address, value)
	}
	return nil, nil
}

func (m *MockModbusClient) Close() {
	m.mu.Lock()
	m.CloseCalls++
	m.mu.Unlock()

	if m.CloseFunc != nil {
		m.CloseFunc()
	}
}

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
var _ controllers.MessagePublisher = (*MockMessagePublisher)(nil)

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

// MockMetricsCollector is a mock implementation of the MetricsCollector interface for testing.
type MockMetricsCollector struct {
	mu sync.RWMutex

	// Function fields that can be set to customize behavior in tests
	IncrementFailuresFunc      func()
	IncrementWriteFailuresFunc func()
	SetMetricsFunc             func(status interface{})

	// Call tracking
	FailuresCount      int
	WriteFailuresCount int
	SetMetricsCalls    []interface{}
}

// Verify MockMetricsCollector implements MetricsCollector
var _ controllers.MetricsCollector = (*MockMetricsCollector)(nil)

func (m *MockMetricsCollector) IncrementFailures() {
	m.mu.Lock()
	m.FailuresCount++
	m.mu.Unlock()

	if m.IncrementFailuresFunc != nil {
		m.IncrementFailuresFunc()
	}
}

func (m *MockMetricsCollector) IncrementWriteFailures() {
	m.mu.Lock()
	m.WriteFailuresCount++
	m.mu.Unlock()

	if m.IncrementWriteFailuresFunc != nil {
		m.IncrementWriteFailuresFunc()
	}
}

func (m *MockMetricsCollector) SetMetrics(status interface{}) {
	m.mu.Lock()
	m.SetMetricsCalls = append(m.SetMetricsCalls, status)
	m.mu.Unlock()

	if m.SetMetricsFunc != nil {
		m.SetMetricsFunc(status)
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

