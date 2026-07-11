package voltgo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lumberbarons/voltgo/battery"
)

func testBatteryStatus() *battery.Status {
	return &battery.Status{
		Voltage:      13.28,
		Current:      -2.5,
		SOC:          87,
		SOH:          100,
		Temperature:  21.5,
		Temperatures: []int{21, 22},
		CellCount:    4,
		Cells: []battery.Cell{
			{Index: 0, Voltage: 3.321},
			{Index: 1, Voltage: 3.319},
			{Index: 2, Voltage: 3.322},
			{Index: 3, Voltage: 3.318},
		},
	}
}

func TestCollector_GetStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("connects lazily and maps status", func(t *testing.T) {
		var connectCtxHadDeadline bool
		mockBattery := &MockBatteryClient{
			GetStatusFunc: func(_ context.Context) (*battery.Status, error) {
				return testBatteryStatus(), nil
			},
		}
		mockConnector := &MockBatteryConnector{
			ConnectFunc: func(ctx context.Context, _ string) (BatteryClient, error) {
				_, connectCtxHadDeadline = ctx.Deadline()
				return mockBattery, nil
			},
		}

		collector := NewCollector(mockConnector, "AA:BB:CC:DD:EE:FF", 10*time.Second)
		status, err := collector.GetStatus(ctx)

		if err != nil {
			t.Fatalf("GetStatus() error = %v", err)
		}
		if status == nil {
			t.Fatal("GetStatus() returned nil status")
		}

		if len(mockConnector.ConnectCalls) != 1 {
			t.Errorf("Connect calls = %d, want 1", len(mockConnector.ConnectCalls))
		}
		if mockConnector.ConnectCalls[0] != "AA:BB:CC:DD:EE:FF" {
			t.Errorf("Connect address = %s, want AA:BB:CC:DD:EE:FF", mockConnector.ConnectCalls[0])
		}
		if !connectCtxHadDeadline {
			t.Error("Connect context should have a deadline from connectTimeout")
		}

		if status.Voltage != 13.28 {
			t.Errorf("Voltage = %v, want 13.28", status.Voltage)
		}
		if status.Current != -2.5 {
			t.Errorf("Current = %v, want -2.5", status.Current)
		}
		if status.SOC != 87 {
			t.Errorf("SOC = %d, want 87", status.SOC)
		}
		if status.SOH != 100 {
			t.Errorf("SOH = %d, want 100", status.SOH)
		}
		if status.Temperature != 21.5 {
			t.Errorf("Temperature = %v, want 21.5", status.Temperature)
		}
		if len(status.Temperatures) != 2 || status.Temperatures[0] != 21 || status.Temperatures[1] != 22 {
			t.Errorf("Temperatures = %v, want [21 22]", status.Temperatures)
		}
		if status.CellCount != 4 {
			t.Errorf("CellCount = %d, want 4", status.CellCount)
		}
		if len(status.Cells) != 4 {
			t.Fatalf("Cells length = %d, want 4", len(status.Cells))
		}
		if status.Cells[2].Index != 2 || status.Cells[2].Voltage != 3.322 {
			t.Errorf("Cells[2] = %+v, want {Index: 2, Voltage: 3.322}", status.Cells[2])
		}
		if status.Timestamp == 0 {
			t.Error("Timestamp should be set")
		}
		if status.CollectionTime < 0 {
			t.Errorf("CollectionTime = %v, should not be negative", status.CollectionTime)
		}
	})

	t.Run("reuses connection while connected", func(t *testing.T) {
		mockBattery := &MockBatteryClient{
			GetStatusFunc: func(_ context.Context) (*battery.Status, error) {
				return testBatteryStatus(), nil
			},
		}
		mockConnector := &MockBatteryConnector{
			ConnectFunc: func(_ context.Context, _ string) (BatteryClient, error) {
				return mockBattery, nil
			},
		}

		collector := NewCollector(mockConnector, "AA:BB:CC:DD:EE:FF", 10*time.Second)

		for i := range 3 {
			if _, err := collector.GetStatus(ctx); err != nil {
				t.Fatalf("GetStatus() call %d error = %v", i, err)
			}
		}

		if len(mockConnector.ConnectCalls) != 1 {
			t.Errorf("Connect calls = %d, want 1 (connection should be reused)", len(mockConnector.ConnectCalls))
		}
		if mockBattery.GetStatusCalls != 3 {
			t.Errorf("GetStatus calls = %d, want 3", mockBattery.GetStatusCalls)
		}
	})

	t.Run("reconnects when connection is lost", func(t *testing.T) {
		connected := true
		staleBattery := &MockBatteryClient{
			GetStatusFunc: func(_ context.Context) (*battery.Status, error) {
				return testBatteryStatus(), nil
			},
			IsConnectedFunc: func() bool { return connected },
		}
		freshBattery := &MockBatteryClient{
			GetStatusFunc: func(_ context.Context) (*battery.Status, error) {
				return testBatteryStatus(), nil
			},
		}

		batteries := []BatteryClient{staleBattery, freshBattery}
		mockConnector := &MockBatteryConnector{}
		mockConnector.ConnectFunc = func(_ context.Context, _ string) (BatteryClient, error) {
			return batteries[len(mockConnector.ConnectCalls)-1], nil
		}

		collector := NewCollector(mockConnector, "AA:BB:CC:DD:EE:FF", 10*time.Second)

		if _, err := collector.GetStatus(ctx); err != nil {
			t.Fatalf("first GetStatus() error = %v", err)
		}

		// Battery now reports disconnected; the next cycle must drop it
		// and reconnect
		connected = false
		if _, err := collector.GetStatus(ctx); err != nil {
			t.Fatalf("second GetStatus() error = %v", err)
		}

		if len(mockConnector.ConnectCalls) != 2 {
			t.Errorf("Connect calls = %d, want 2 (should reconnect after connection loss)", len(mockConnector.ConnectCalls))
		}
		if staleBattery.DisconnectCalls != 1 {
			t.Errorf("stale battery Disconnect calls = %d, want 1", staleBattery.DisconnectCalls)
		}
		if freshBattery.GetStatusCalls != 1 {
			t.Errorf("fresh battery GetStatus calls = %d, want 1", freshBattery.GetStatusCalls)
		}
	})

	t.Run("read error drops connection and next cycle reconnects", func(t *testing.T) {
		failingBattery := &MockBatteryClient{
			GetStatusFunc: func(_ context.Context) (*battery.Status, error) {
				return nil, errors.New("BLE read timeout")
			},
		}
		workingBattery := &MockBatteryClient{
			GetStatusFunc: func(_ context.Context) (*battery.Status, error) {
				return testBatteryStatus(), nil
			},
		}

		batteries := []BatteryClient{failingBattery, workingBattery}
		mockConnector := &MockBatteryConnector{}
		mockConnector.ConnectFunc = func(_ context.Context, _ string) (BatteryClient, error) {
			return batteries[len(mockConnector.ConnectCalls)-1], nil
		}

		collector := NewCollector(mockConnector, "AA:BB:CC:DD:EE:FF", 10*time.Second)

		if _, err := collector.GetStatus(ctx); err == nil {
			t.Fatal("first GetStatus() should return the read error")
		}
		if failingBattery.DisconnectCalls != 1 {
			t.Errorf("failing battery Disconnect calls = %d, want 1 (connection should be dropped on read error)", failingBattery.DisconnectCalls)
		}

		status, err := collector.GetStatus(ctx)
		if err != nil {
			t.Fatalf("second GetStatus() error = %v", err)
		}
		if status == nil {
			t.Fatal("second GetStatus() returned nil status")
		}
		if len(mockConnector.ConnectCalls) != 2 {
			t.Errorf("Connect calls = %d, want 2 (should reconnect after read error)", len(mockConnector.ConnectCalls))
		}
	})

	t.Run("connect error is returned", func(t *testing.T) {
		mockConnector := &MockBatteryConnector{
			ConnectFunc: func(_ context.Context, _ string) (BatteryClient, error) {
				return nil, errors.New("device not found")
			},
		}

		collector := NewCollector(mockConnector, "AA:BB:CC:DD:EE:FF", 10*time.Second)

		if _, err := collector.GetStatus(ctx); err == nil {
			t.Fatal("GetStatus() should return the connect error")
		}

		// A failed connect leaves no battery behind; the next cycle retries
		if _, err := collector.GetStatus(ctx); err == nil {
			t.Fatal("second GetStatus() should also return the connect error")
		}
		if len(mockConnector.ConnectCalls) != 2 {
			t.Errorf("Connect calls = %d, want 2 (should retry connect each cycle)", len(mockConnector.ConnectCalls))
		}
	})
}

func TestCollector_GetInfo(t *testing.T) {
	ctx := context.Background()

	t.Run("connects and maps info", func(t *testing.T) {
		mockBattery := &MockBatteryClient{
			GetInfoFunc: func(_ context.Context) (*battery.Info, error) {
				return &battery.Info{
					Chemistry:      "LiFePO4",
					NominalVoltage: 12.8,
					CapacityAh:     100,
					DeviceStrings:  []string{"VOLTGO-100", "HW1.2"},
				}, nil
			},
		}
		mockConnector := &MockBatteryConnector{
			ConnectFunc: func(_ context.Context, _ string) (BatteryClient, error) {
				return mockBattery, nil
			},
		}

		collector := NewCollector(mockConnector, "AA:BB:CC:DD:EE:FF", 10*time.Second)
		info, err := collector.GetInfo(ctx)

		if err != nil {
			t.Fatalf("GetInfo() error = %v", err)
		}
		if info.Chemistry != "LiFePO4" {
			t.Errorf("Chemistry = %s, want LiFePO4", info.Chemistry)
		}
		if info.NominalVoltage != 12.8 {
			t.Errorf("NominalVoltage = %v, want 12.8", info.NominalVoltage)
		}
		if info.CapacityAh != 100 {
			t.Errorf("CapacityAh = %v, want 100", info.CapacityAh)
		}
		if len(info.DeviceStrings) != 2 || info.DeviceStrings[0] != "VOLTGO-100" {
			t.Errorf("DeviceStrings = %v, want [VOLTGO-100 HW1.2]", info.DeviceStrings)
		}
	})

	t.Run("read error drops connection", func(t *testing.T) {
		mockBattery := &MockBatteryClient{
			GetInfoFunc: func(_ context.Context) (*battery.Info, error) {
				return nil, errors.New("BLE read timeout")
			},
		}
		mockConnector := &MockBatteryConnector{
			ConnectFunc: func(_ context.Context, _ string) (BatteryClient, error) {
				return mockBattery, nil
			},
		}

		collector := NewCollector(mockConnector, "AA:BB:CC:DD:EE:FF", 10*time.Second)

		if _, err := collector.GetInfo(ctx); err == nil {
			t.Fatal("GetInfo() should return the read error")
		}
		if mockBattery.DisconnectCalls != 1 {
			t.Errorf("Disconnect calls = %d, want 1", mockBattery.DisconnectCalls)
		}
	})
}

func TestCollector_Close(t *testing.T) {
	ctx := context.Background()

	t.Run("disconnects battery and closes connector", func(t *testing.T) {
		mockBattery := &MockBatteryClient{
			GetStatusFunc: func(_ context.Context) (*battery.Status, error) {
				return testBatteryStatus(), nil
			},
		}
		mockConnector := &MockBatteryConnector{
			ConnectFunc: func(_ context.Context, _ string) (BatteryClient, error) {
				return mockBattery, nil
			},
		}

		collector := NewCollector(mockConnector, "AA:BB:CC:DD:EE:FF", 10*time.Second)
		if _, err := collector.GetStatus(ctx); err != nil {
			t.Fatalf("GetStatus() error = %v", err)
		}

		if err := collector.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		if mockBattery.DisconnectCalls != 1 {
			t.Errorf("Disconnect calls = %d, want 1", mockBattery.DisconnectCalls)
		}
		if mockConnector.CloseCalls != 1 {
			t.Errorf("connector Close calls = %d, want 1", mockConnector.CloseCalls)
		}
	})

	t.Run("closes connector when never connected", func(t *testing.T) {
		mockConnector := &MockBatteryConnector{}

		collector := NewCollector(mockConnector, "AA:BB:CC:DD:EE:FF", 10*time.Second)
		if err := collector.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		if mockConnector.CloseCalls != 1 {
			t.Errorf("connector Close calls = %d, want 1", mockConnector.CloseCalls)
		}
	})
}

func TestConfiguration_GetConnectTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout string
		want    time.Duration
	}{
		{"default when empty", "", 30 * time.Second},
		{"parses valid duration", "45s", 45 * time.Second},
		{"default on invalid duration", "bogus", 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Configuration{ConnectTimeout: tt.timeout}
			if got := c.GetConnectTimeout(); got != tt.want {
				t.Errorf("GetConnectTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfiguration_Validate(t *testing.T) {
	valid := &Configuration{ConnectTimeout: "10s"}
	if err := valid.Validate(); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}

	empty := &Configuration{}
	if err := empty.Validate(); err != nil {
		t.Errorf("Validate() error = %v, want nil for empty timeout", err)
	}

	invalid := &Configuration{ConnectTimeout: "bogus"}
	if err := invalid.Validate(); err == nil {
		t.Error("Validate() should return an error for an invalid duration")
	}
}
