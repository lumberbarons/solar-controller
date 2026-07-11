package voltgo

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// Collector owns the BLE connection lifecycle: it connects lazily on first
// collection, reuses the connection while it stays alive, and drops it on
// read errors so the next cycle reconnects.
type Collector struct {
	connector      BatteryConnector
	address        string
	connectTimeout time.Duration

	mu      sync.Mutex
	battery BatteryClient
}

type BatteryStatus struct {
	Timestamp      int64   `json:"timestamp"`
	CollectionTime float64 `json:"collectionTime"`
	Voltage        float64 `json:"voltage"`
	Current        float64 `json:"current"`
	SOC            int     `json:"soc"`
	SOH            int     `json:"soh"`
	Temperature    float64 `json:"temperature"`
	Temperatures   []int   `json:"temperatures"`
	CellCount      int     `json:"cellCount"`
	Cells          []Cell  `json:"cells"`
}

type Cell struct {
	Index   int     `json:"index"`
	Voltage float64 `json:"voltage"`
}

type BatteryInfo struct {
	Chemistry      string   `json:"chemistry"`
	NominalVoltage float64  `json:"nominalVoltage"`
	CapacityAh     float64  `json:"capacityAh"`
	DeviceStrings  []string `json:"deviceStrings"`
}

func NewCollector(connector BatteryConnector, address string, connectTimeout time.Duration) *Collector {
	if connectTimeout <= 0 {
		connectTimeout = defaultConnectTimeout
	}
	return &Collector{
		connector:      connector,
		address:        address,
		connectTimeout: connectTimeout,
	}
}

func (c *Collector) GetStatus(ctx context.Context) (*BatteryStatus, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	startTime := time.Now()
	log.Debug("Starting voltgo GetStatus collection")

	bat, err := c.connectLocked(ctx)
	if err != nil {
		return nil, err
	}

	status, err := bat.GetStatus(ctx)
	if err != nil {
		c.dropConnectionLocked()
		return nil, fmt.Errorf("failed to read voltgo battery status: %w", err)
	}

	result := &BatteryStatus{
		Timestamp:      startTime.Unix(),
		Voltage:        status.Voltage,
		Current:        status.Current,
		SOC:            status.SOC,
		SOH:            status.SOH,
		Temperature:    status.Temperature,
		Temperatures:   status.Temperatures,
		CellCount:      status.CellCount,
		Cells:          make([]Cell, 0, len(status.Cells)),
		CollectionTime: 0,
	}
	for _, cell := range status.Cells {
		result.Cells = append(result.Cells, Cell{Index: cell.Index, Voltage: cell.Voltage})
	}

	result.CollectionTime = time.Since(startTime).Seconds()
	log.Debugf("voltgo GetStatus completed in %.3fs - %.2fV/%.2fA, SOC %d%%, SOH %d%%, %.1f°C, %d cells",
		result.CollectionTime, result.Voltage, result.Current, result.SOC, result.SOH,
		result.Temperature, result.CellCount)

	return result, nil
}

func (c *Collector) GetInfo(ctx context.Context) (*BatteryInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bat, err := c.connectLocked(ctx)
	if err != nil {
		return nil, err
	}

	info, err := bat.GetInfo(ctx)
	if err != nil {
		c.dropConnectionLocked()
		return nil, fmt.Errorf("failed to read voltgo battery info: %w", err)
	}

	return &BatteryInfo{
		Chemistry:      info.Chemistry,
		NominalVoltage: info.NominalVoltage,
		CapacityAh:     info.CapacityAh,
		DeviceStrings:  info.DeviceStrings,
	}, nil
}

// Close drops any active battery connection and releases the BLE adapter.
func (c *Collector) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.dropConnectionLocked()
	return c.connector.Close()
}

// connectLocked returns the current battery connection, establishing or
// re-establishing it as needed. Callers must hold c.mu.
func (c *Collector) connectLocked(ctx context.Context) (BatteryClient, error) {
	if c.battery != nil {
		if c.battery.IsConnected() {
			return c.battery, nil
		}
		log.Warnf("voltgo battery %s connection lost, reconnecting", c.address)
		c.dropConnectionLocked()
	}

	connectCtx, cancel := context.WithTimeout(ctx, c.connectTimeout)
	defer cancel()

	log.Debugf("connecting to voltgo battery %s", c.address)
	bat, err := c.connector.Connect(connectCtx, c.address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to voltgo battery %s: %w", c.address, err)
	}

	log.Infof("connected to voltgo battery %s", c.address)
	c.battery = bat
	return bat, nil
}

// dropConnectionLocked disconnects and forgets the current battery so the
// next collection cycle reconnects. Callers must hold c.mu.
func (c *Collector) dropConnectionLocked() {
	if c.battery == nil {
		return
	}
	if err := c.battery.Disconnect(); err != nil {
		log.Debugf("error disconnecting voltgo battery %s: %v", c.address, err)
	}
	c.battery = nil
}
