package publishers

import (
	"testing"

	"github.com/lumberbarons/solar-controller/internal/config"
	"github.com/lumberbarons/solar-controller/internal/controllers"
	"github.com/lumberbarons/solar-controller/internal/file"
	"github.com/lumberbarons/solar-controller/internal/mqtt"
	"github.com/lumberbarons/solar-controller/internal/remotewrite"
	"github.com/lumberbarons/solar-controller/internal/sns"
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
				Mqtt: mqtt.Configuration{
					Enabled:     true,
					Host:        "", // Empty host prevents connection attempt
					TopicPrefix: "test",
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
				Mqtt: mqtt.Configuration{
					Enabled: false,
				},
				Solace: solace.Configuration{
					Enabled:     true,
					Host:        "", // Empty host prevents connection attempt
					VpnName:     "default",
					TopicPrefix: "test",
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
			name: "Both MQTT and Solace enabled - returns MultiPublisher",
			config: config.SolarControllerConfiguration{
				Mqtt: mqtt.Configuration{
					Enabled:     true,
					Host:        "", // Empty host prevents connection attempt
					TopicPrefix: "test",
				},
				Solace: solace.Configuration{
					Enabled:     true,
					Host:        "", // Empty host prevents connection attempt
					VpnName:     "default",
					TopicPrefix: "test",
				},
			},
			wantErr: false,
			check: func(t *testing.T, pub interface{}) {
				if _, ok := pub.(*MultiPublisher); !ok {
					t.Errorf("expected *MultiPublisher, got %T", pub)
				}
			},
		},
		{
			name: "SNS enabled but no topic ARN - returns empty SNS publisher",
			config: config.SolarControllerConfiguration{
				SNS: sns.Configuration{
					Enabled:  true,
					TopicArn: "", // Empty ARN prevents connection
					Region:   "us-east-1",
				},
			},
			wantErr: false,
			check: func(t *testing.T, pub interface{}) {
				if _, ok := pub.(*sns.Publisher); !ok {
					t.Errorf("expected *sns.Publisher, got %T", pub)
				}
			},
		},
		{
			name: "File enabled but no filename - returns empty File publisher",
			config: config.SolarControllerConfiguration{
				File: file.Configuration{
					Enabled:  true,
					Filename: "", // Empty filename prevents creation
				},
			},
			wantErr: false,
			check: func(t *testing.T, pub interface{}) {
				if _, ok := pub.(*file.Publisher); !ok {
					t.Errorf("expected *file.Publisher, got %T", pub)
				}
			},
		},
		{
			name: "RemoteWrite disabled - returns empty RemoteWrite publisher",
			config: config.SolarControllerConfiguration{
				RemoteWrite: remotewrite.Configuration{
					Enabled: true,
					URL:     "http://localhost:9090/api/v1/write",
				},
			},
			wantErr: false,
			check: func(t *testing.T, pub interface{}) {
				if _, ok := pub.(*remotewrite.Publisher); !ok {
					t.Errorf("expected *remotewrite.Publisher, got %T", pub)
				}
			},
		},
		{
			name: "Three publishers enabled - returns MultiPublisher with 3 publishers",
			config: config.SolarControllerConfiguration{
				Mqtt: mqtt.Configuration{
					Enabled:     true,
					Host:        "", // Empty host prevents connection
					TopicPrefix: "test",
				},
				Solace: solace.Configuration{
					Enabled:     true,
					Host:        "", // Empty host prevents connection
					VpnName:     "default",
					TopicPrefix: "test",
				},
				SNS: sns.Configuration{
					Enabled:  true,
					TopicArn: "", // Empty ARN prevents connection
					Region:   "us-east-1",
				},
			},
			wantErr: false,
			check: func(t *testing.T, pub interface{}) {
				mp, ok := pub.(*MultiPublisher)
				if !ok {
					t.Fatalf("expected *MultiPublisher, got %T", pub)
				}
				if len(mp.publishers) != 3 {
					t.Errorf("expected 3 publishers, got %d", len(mp.publishers))
				}
			},
		},
		{
			name: "RemoteWrite enabled with empty URL - returns error",
			config: config.SolarControllerConfiguration{
				RemoteWrite: remotewrite.Configuration{
					Enabled: true,
					URL:     "", // Missing required URL
				},
			},
			wantErr: true,
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

func TestNoOpPublisher(t *testing.T) {
	// Verify NoOpPublisher satisfies MessagePublisher interface at compile time
	var pub controllers.MessagePublisher = &NoOpPublisher{}

	// Should not panic when used as MessagePublisher
	pub.Publish("test", "payload")
	pub.Close()

	// Verify the concrete type is preserved after operations
	if _, ok := pub.(*NoOpPublisher); !ok {
		t.Error("Expected publisher to remain a *NoOpPublisher")
	}
}
