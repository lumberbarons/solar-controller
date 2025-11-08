package publishers

import (
	"testing"

	"github.com/lumberbarons/solar-controller/internal/config"
	"github.com/lumberbarons/solar-controller/internal/mqtt"
	"github.com/lumberbarons/solar-controller/internal/solace"
)

func TestNewPublisher(t *testing.T) {
	tests := []struct {
		name    string
		config  config.SolarControllerConfiguration
		wantErr bool
		check   func(*testing.T, interface{})
	}{
		{
			name: "MQTT enabled but no host - returns empty MQTT publisher",
			config: config.SolarControllerConfiguration{
				TopicPrefix: "test",
				Mqtt: mqtt.Configuration{
					Enabled: true,
					Host:    "", // Empty host prevents connection attempt
				},
				Solace: solace.Configuration{
					Enabled: false,
				},
			},
			wantErr: false,
			check: func(t *testing.T, pub interface{}) {
				if _, ok := pub.(*mqtt.Publisher); !ok {
					t.Errorf("expected *mqtt.Publisher, got %T", pub)
				}
			},
		},
		{
			name: "Solace enabled but no host - returns empty Solace publisher",
			config: config.SolarControllerConfiguration{
				TopicPrefix: "test",
				Mqtt: mqtt.Configuration{
					Enabled: false,
				},
				Solace: solace.Configuration{
					Enabled: true,
					Host:    "", // Empty host prevents connection attempt
					VpnName: "default",
				},
			},
			wantErr: false,
			check: func(t *testing.T, pub interface{}) {
				if _, ok := pub.(*solace.Publisher); !ok {
					t.Errorf("expected *solace.Publisher, got %T", pub)
				}
			},
		},
		{
			name: "Neither enabled - returns NoOpPublisher",
			config: config.SolarControllerConfiguration{
				Mqtt: mqtt.Configuration{
					Enabled: false,
				},
				Solace: solace.Configuration{
					Enabled: false,
				},
			},
			wantErr: false,
			check: func(t *testing.T, pub interface{}) {
				if _, ok := pub.(*NoOpPublisher); !ok {
					t.Errorf("expected *NoOpPublisher, got %T", pub)
				}
			},
		},
		{
			name: "Both enabled - returns error",
			config: config.SolarControllerConfiguration{
				TopicPrefix: "test",
				Mqtt: mqtt.Configuration{
					Enabled: true,
					Host:    "",
				},
				Solace: solace.Configuration{
					Enabled: true,
					Host:    "",
					VpnName: "default",
				},
			},
			wantErr: true,
		},
		{
			name: "MQTT disabled, Solace disabled - returns no-op publisher",
			config: config.SolarControllerConfiguration{
				Mqtt: mqtt.Configuration{
					Enabled: false,
				},
				Solace: solace.Configuration{
					Enabled: false,
				},
			},
			wantErr: false,
			check: func(t *testing.T, pub interface{}) {
				if _, ok := pub.(*NoOpPublisher); !ok {
					t.Errorf("expected *NoOpPublisher, got %T", pub)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pub, err := NewPublisher(&tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("NewPublisher() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewPublisher() unexpected error: %v", err)
				return
			}

			if pub == nil {
				t.Error("NewPublisher() returned nil publisher")
				return
			}

			if tt.check != nil {
				tt.check(t, pub)
			}
		})
	}
}

func TestNoOpPublisher(_ *testing.T) {
	pub := &NoOpPublisher{}

	// Should not panic
	pub.Publish("test", "payload")
	pub.Close()

	// Test passed if we got here without panic
}
