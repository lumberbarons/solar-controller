package epever

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	log "github.com/sirupsen/logrus"

	"github.com/lumberbarons/modbus"
)

type ModbusClient struct {
	handler *modbus.RTUClientHandler
	client  modbus.Client
	lock    sync.Mutex
}

const retryAttempts = 2
const retryDelay = 5 * time.Second

func NewModbusClient(serialPort string) (*ModbusClient, error) {
	handler := modbus.NewRTUClientHandler(serialPort)

	handler.BaudRate = 115200
	handler.DataBits = 8
	handler.Parity = modbus.NoParity
	handler.StopBits = 1
	handler.SlaveID = 1
	handler.Timeout = 5 * time.Second

	err := handler.Connect()

	if err != nil {
		return nil, fmt.Errorf("failed to connect to epever: %w", err)
	}

	client := modbus.NewClient(handler)

	return &ModbusClient{handler: handler, client: client}, nil
}

func (c *ModbusClient) Close() {
	if c.handler != nil {
		c.handler.Close()
	}
}

func (c *ModbusClient) ReadInputRegisters(ctx context.Context, address, quantity uint16) ([]byte, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	var value []byte

	err := retry.Do(
		func() error {
			var retryErr error
			value, retryErr = c.client.ReadInputRegisters(ctx, address, quantity)
			return retryErr
		},
		retry.Attempts(retryAttempts),
		retry.Delay(retryDelay),
		retry.OnRetry(func(n uint, err error) {
			log.Warnf("ReadInputRegisters address %d retry #%d: %s\n", address, n, err)
		}),
		retry.Context(ctx),
	)

	return value, err
}

func (c *ModbusClient) ReadHoldingRegisters(ctx context.Context, address, quantity uint16) ([]byte, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	var value []byte

	err := retry.Do(
		func() error {
			var retryErr error
			value, retryErr = c.client.ReadHoldingRegisters(ctx, address, quantity)
			return retryErr
		},
		retry.Attempts(retryAttempts),
		retry.Delay(retryDelay),
		retry.OnRetry(func(n uint, err error) {
			log.Warnf("ReadHoldingRegisters address %d retry #%d: %s\n", address, n, err)
		}),
		retry.Context(ctx),
	)

	return value, err
}

func (c *ModbusClient) ReadCoils(ctx context.Context, address, quantity uint16) ([]byte, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	var value []byte

	err := retry.Do(
		func() error {
			var retryErr error
			value, retryErr = c.client.ReadCoils(ctx, address, quantity)
			return retryErr
		},
		retry.Attempts(retryAttempts),
		retry.Delay(retryDelay),
		retry.OnRetry(func(n uint, err error) {
			log.Warnf("ReadCoils address %d retry #%d: %s\n", address, n, err)
		}),
		retry.Context(ctx),
	)

	return value, err
}

func (c *ModbusClient) ReadDiscreteInputs(ctx context.Context, address, quantity uint16) ([]byte, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	var value []byte

	err := retry.Do(
		func() error {
			var retryErr error
			value, retryErr = c.client.ReadDiscreteInputs(ctx, address, quantity)
			return retryErr
		},
		retry.Attempts(retryAttempts),
		retry.Delay(retryDelay),
		retry.OnRetry(func(n uint, err error) {
			log.Warnf("ReadDiscreteInputs address %d retry #%d: %s\n", address, n, err)
		}),
		retry.Context(ctx),
	)

	return value, err
}

func (c *ModbusClient) WriteSingleRegister(ctx context.Context, address, value uint16) (results []byte, err error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	var result []byte

	err = retry.Do(
		func() error {
			var retryErr error
			result, retryErr = c.client.WriteSingleRegister(ctx, address, value)
			return retryErr
		},
		retry.Attempts(retryAttempts),
		retry.Delay(retryDelay),
		retry.OnRetry(func(n uint, err error) {
			log.Warnf("WriteSingleRegister address %d retry #%d: %s\n", address, n, err)
		}),
		retry.Context(ctx),
	)

	return result, err
}

func (c *ModbusClient) WriteMultipleRegisters(ctx context.Context, address, quantity uint16, value []byte) (results []byte, err error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	var result []byte

	err = retry.Do(
		func() error {
			var retryErr error
			result, retryErr = c.client.WriteMultipleRegisters(ctx, address, quantity, value)
			return retryErr
		},
		retry.Attempts(retryAttempts),
		retry.Delay(retryDelay),
		retry.OnRetry(func(n uint, err error) {
			log.Warnf("WriteMultipleRegisters address %d retry #%d: %s\n", address, n, err)
		}),
		retry.Context(ctx),
	)

	return result, err
}
