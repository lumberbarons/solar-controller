package voltgo

import (
	"strings"
	"testing"
	"time"
)

func TestResolveBatteries(t *testing.T) {
	tests := []struct {
		name    string
		config  Configuration
		want    []BatteryConfiguration
		wantErr string
	}{
		{
			name:   "no batteries resolves to an empty list",
			config: Configuration{Enabled: true},
			want:   []BatteryConfiguration{},
		},
		{
			// The deprecated singular form is what every existing
			// deployment has in its config file today.
			name:   "singular address is treated as one battery",
			config: Configuration{Enabled: true, Address: "AA:BB:CC:DD:EE:FF"},
			want:   []BatteryConfiguration{{ID: "aa-bb-cc-dd-ee-ff", Address: "AA:BB:CC:DD:EE:FF"}},
		},
		{
			name: "battery without an id gets one derived from its address",
			config: Configuration{
				Enabled:   true,
				Batteries: []BatteryConfiguration{{Address: "A4:C1:37:43:A4:33"}},
			},
			want: []BatteryConfiguration{{ID: "a4-c1-37-43-a4-33", Address: "A4:C1:37:43:A4:33"}},
		},
		{
			name: "configured ids are preserved in order",
			config: Configuration{
				Enabled: true,
				Batteries: []BatteryConfiguration{
					{ID: "bank-a", Address: "A4:C1:37:43:A4:33"},
					{ID: "bank-b", Address: "A4:C1:37:43:A4:42"},
					{ID: "bank-c", Address: "A4:C1:37:23:A4:3F"},
					{ID: "bank-d", Address: "A4:C1:37:23:A4:40"},
				},
			},
			want: []BatteryConfiguration{
				{ID: "bank-a", Address: "A4:C1:37:43:A4:33"},
				{ID: "bank-b", Address: "A4:C1:37:43:A4:42"},
				{ID: "bank-c", Address: "A4:C1:37:23:A4:3F"},
				{ID: "bank-d", Address: "A4:C1:37:23:A4:40"},
			},
		},
		{
			name: "address and batteries together is an error",
			config: Configuration{
				Enabled:   true,
				Address:   "AA:BB:CC:DD:EE:FF",
				Batteries: []BatteryConfiguration{{Address: "A4:C1:37:43:A4:33"}},
			},
			wantErr: "mutually exclusive",
		},
		{
			name: "battery without an address is an error",
			config: Configuration{
				Enabled:   true,
				Batteries: []BatteryConfiguration{{ID: "bank-a"}},
			},
			wantErr: "address is required",
		},
		{
			name: "duplicate id is an error",
			config: Configuration{
				Enabled: true,
				Batteries: []BatteryConfiguration{
					{ID: "bank-a", Address: "A4:C1:37:43:A4:33"},
					{ID: "bank-a", Address: "A4:C1:37:43:A4:42"},
				},
			},
			wantErr: `id "bank-a" is configured more than once`,
		},
		{
			// Two batteries sharing one address would be two runners
			// fighting over the same BLE connection.
			name: "duplicate address is an error",
			config: Configuration{
				Enabled: true,
				Batteries: []BatteryConfiguration{
					{ID: "bank-a", Address: "A4:C1:37:43:A4:33"},
					{ID: "bank-b", Address: "A4:C1:37:43:A4:33"},
				},
			},
			wantErr: "is configured more than once",
		},
		{
			// BlueZ lowercases addresses; the datasheet uppercases them.
			// Comparing case-sensitively would let the same battery in twice.
			name: "duplicate address differing only in case is an error",
			config: Configuration{
				Enabled: true,
				Batteries: []BatteryConfiguration{
					{ID: "bank-a", Address: "A4:C1:37:43:A4:33"},
					{ID: "bank-b", Address: "a4:c1:37:43:a4:33"},
				},
			},
			wantErr: "is configured more than once",
		},
		{
			// An id with a slash would split into two path segments and two
			// topic levels; the derived-from-address default cannot produce
			// one, but a hand-written id can.
			name: "id with a path separator is an error",
			config: Configuration{
				Enabled:   true,
				Batteries: []BatteryConfiguration{{ID: "bank/a", Address: "A4:C1:37:43:A4:33"}},
			},
			wantErr: "id must match",
		},
		{
			name: "id with an MQTT wildcard character is an error",
			config: Configuration{
				Enabled:   true,
				Batteries: []BatteryConfiguration{{ID: "bank+a", Address: "A4:C1:37:43:A4:33"}},
			},
			wantErr: "id must match",
		},
		{
			name: "id with uppercase is an error",
			config: Configuration{
				Enabled:   true,
				Batteries: []BatteryConfiguration{{ID: "BankA", Address: "A4:C1:37:43:A4:33"}},
			},
			wantErr: "id must match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.config.ResolveBatteries()

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("ResolveBatteries() error = nil, want an error containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("ResolveBatteries() error = %q, want it to contain %q", err.Error(), tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("ResolveBatteries() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ResolveBatteries() = %+v, want %+v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("battery %d = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestConfigurationValidate(t *testing.T) {
	t.Run("rejects an unparseable connect timeout", func(t *testing.T) {
		config := Configuration{Enabled: true, Address: "AA:BB:CC:DD:EE:FF", ConnectTimeout: "nonsense"}
		if err := config.Validate(); err == nil {
			t.Error("Validate() error = nil, want a parse error")
		}
	})

	t.Run("surfaces battery list errors", func(t *testing.T) {
		config := Configuration{
			Enabled: true,
			Batteries: []BatteryConfiguration{
				{ID: "bank-a", Address: "A4:C1:37:43:A4:33"},
				{ID: "bank-a", Address: "A4:C1:37:43:A4:42"},
			},
		}
		if err := config.Validate(); err == nil {
			t.Error("Validate() error = nil, want a duplicate id error")
		}
	})

	t.Run("accepts a valid multi-battery configuration", func(t *testing.T) {
		config := Configuration{
			Enabled:        true,
			ConnectTimeout: "45s",
			Batteries: []BatteryConfiguration{
				{ID: "bank-a", Address: "A4:C1:37:43:A4:33"},
				{ID: "bank-b", Address: "A4:C1:37:43:A4:42"},
			},
		}
		if err := config.Validate(); err != nil {
			t.Errorf("Validate() error = %v", err)
		}
		if got := config.GetConnectTimeout(); got != 45*time.Second {
			t.Errorf("GetConnectTimeout() = %s, want 45s", got)
		}
	})
}

func TestGetConnectTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout string
		want    time.Duration
	}{
		{"empty falls back to the default", "", defaultConnectTimeout},
		{"unparseable falls back to the default", "not-a-duration", defaultConnectTimeout},
		{"parseable is used as configured", "15s", 15 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Configuration{ConnectTimeout: tt.timeout}
			if got := config.GetConnectTimeout(); got != tt.want {
				t.Errorf("GetConnectTimeout() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestDefaultBatteryID(t *testing.T) {
	if got := DefaultBatteryID("A4:C1:37:43:A4:33"); got != "a4-c1-37-43-a4-33" {
		t.Errorf("DefaultBatteryID() = %q, want %q", got, "a4-c1-37-43-a4-33")
	}
	// The derived id has to satisfy the same pattern a configured id does,
	// or a battery configured without an id could never start.
	if !batteryIDPattern.MatchString(DefaultBatteryID("A4:C1:37:43:A4:33")) {
		t.Error("a derived id must satisfy batteryIDPattern")
	}
}
