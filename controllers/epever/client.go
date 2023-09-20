package epever

import (
	"fmt"
	"github.com/goburrow/modbus"
	"sync"
	"time"
)

type ModbusClient struct {
	handler *modbus.RTUClientHandler
	client  modbus.Client
	lock    sync.Mutex
}

func NewModbusClient(serialPort string) (*ModbusClient, error) {
	handler := modbus.NewRTUClientHandler(serialPort)

	handler.BaudRate = 115200
	handler.DataBits = 8
	handler.Parity = "N"
	handler.StopBits = 1
	handler.SlaveId = 1
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

func (c *ModbusClient) ReadInputRegisters(address uint16, quantity uint16) (results []byte, err error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.client.ReadInputRegisters(address, quantity)
}

func (c *ModbusClient) ReadHoldingRegisters(address uint16, quantity uint16) (results []byte, err error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.client.ReadHoldingRegisters(address, quantity)
}

func (c *ModbusClient) ReadCoils(address uint16, quantity uint16) (results []byte, err error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.client.ReadCoils(address, quantity)
}

func (c *ModbusClient) ReadDiscreteInputs(address uint16, quantity uint16) (results []byte, err error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.client.ReadDiscreteInputs(address, quantity)
}

func (c *ModbusClient) WriteSingleRegister(address uint16, value uint16) (results []byte, err error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.client.WriteSingleRegister(address, value)
}

func (c *ModbusClient) WriteMultipleRegisters(address, quantity uint16, value []byte) (results []byte, err error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.client.WriteMultipleRegisters(address, quantity, value)
}
