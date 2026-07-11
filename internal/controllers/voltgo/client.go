package voltgo

import (
	"context"
	"fmt"

	voltgolib "github.com/lumberbarons/voltgo"
)

// BLEConnector is the production BatteryConnector backed by the voltgo
// library's BLE client.
type BLEConnector struct {
	client *voltgolib.Client
}

// Verify BLEConnector implements BatteryConnector
var _ BatteryConnector = (*BLEConnector)(nil)

func NewBLEConnector() (*BLEConnector, error) {
	client, err := voltgolib.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize BLE client: %w", err)
	}
	return &BLEConnector{client: client}, nil
}

func (c *BLEConnector) Connect(ctx context.Context, address string) (BatteryClient, error) {
	bat, err := c.client.Connect(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to voltgo battery %s: %w", address, err)
	}
	return bat, nil
}

func (c *BLEConnector) Close() error {
	return c.client.Close()
}
