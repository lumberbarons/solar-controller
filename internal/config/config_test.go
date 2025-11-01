package config

import (
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
		errMsg  string
		check   func(*testing.T, Config)
	}{
		{
			name: "valid configuration with all fields",
			yaml: `
solarController:
  httpPort: 8080
  mqtt:
    enabled: true
    host: mqtt://localhost:1883
    username: user
    password: pass
    topicPrefix: solar/metrics
  epever:
    enabled: true
    serialPort: /dev/ttyUSB0
    publishPeriod: 60
`,
			wantErr: false,
			check: func(t *testing.T, c Config) {
				if c.SolarController.HTTPPort != 8080 {
					t.Errorf("HTTPPort = %d, want 8080", c.SolarController.HTTPPort)
				}
				if !c.SolarController.Mqtt.Enabled {
					t.Error("MQTT should be enabled")
				}
				if c.SolarController.Mqtt.Host != "mqtt://localhost:1883" {
					t.Errorf("MQTT Host = %s, want mqtt://localhost:1883", c.SolarController.Mqtt.Host)
				}
				if c.SolarController.Mqtt.TopicPrefix != "solar/metrics" {
					t.Errorf("MQTT TopicPrefix = %s, want solar/metrics", c.SolarController.Mqtt.TopicPrefix)
				}
				if !c.SolarController.Epever.Enabled {
					t.Error("Epever should be enabled")
				}
				if c.SolarController.Epever.SerialPort != "/dev/ttyUSB0" {
					t.Errorf("Epever SerialPort = %s, want /dev/ttyUSB0", c.SolarController.Epever.SerialPort)
				}
				if c.SolarController.Epever.PublishPeriod != 60 {
					t.Errorf("Epever PublishPeriod = %d, want 60", c.SolarController.Epever.PublishPeriod)
				}
			},
		},
		{
			name: "debug mode enabled in config",
			yaml: `
solarController:
  httpPort: 8080
  debug: true
  epever:
    enabled: false
`,
			wantErr: false,
			check: func(t *testing.T, c Config) {
				if !c.SolarController.Debug {
					t.Error("Debug should be enabled")
				}
			},
		},
		{
			name: "debug mode disabled in config",
			yaml: `
solarController:
  httpPort: 8080
  debug: false
  epever:
    enabled: false
`,
			wantErr: false,
			check: func(t *testing.T, c Config) {
				if c.SolarController.Debug {
					t.Error("Debug should be disabled")
				}
			},
		},
		{
			name: "debug mode not specified defaults to false",
			yaml: `
solarController:
  httpPort: 8080
  epever:
    enabled: false
`,
			wantErr: false,
			check: func(t *testing.T, c Config) {
				if c.SolarController.Debug {
					t.Error("Debug should default to false when not specified")
				}
			},
		},
		{
			name: "minimal valid configuration (epever disabled)",
			yaml: `
solarController:
  httpPort: 8080
  mqtt:
    host: ""
  epever:
    enabled: false
`,
			wantErr: false,
			check: func(t *testing.T, c Config) {
				if c.SolarController.HTTPPort != 8080 {
					t.Errorf("HTTPPort = %d, want 8080", c.SolarController.HTTPPort)
				}
				if c.SolarController.Epever.Enabled {
					t.Error("Epever should be disabled")
				}
			},
		},
		{
			name: "invalid YAML syntax",
			yaml: `
solarController:
  httpPort: 8080
  mqtt:
    host: mqtt://localhost:1883
    invalid syntax here [[[
`,
			wantErr: true,
			errMsg:  "failed to parse YAML",
		},
		{
			name: "invalid HTTP port - zero",
			yaml: `
solarController:
  httpPort: 0
  epever:
    enabled: false
`,
			wantErr: true,
			errMsg:  "invalid HTTP port",
		},
		{
			name: "invalid HTTP port - negative",
			yaml: `
solarController:
  httpPort: -1
  epever:
    enabled: false
`,
			wantErr: true,
			errMsg:  "invalid HTTP port",
		},
		{
			name: "invalid HTTP port - too large",
			yaml: `
solarController:
  httpPort: 99999
  epever:
    enabled: false
`,
			wantErr: true,
			errMsg:  "invalid HTTP port",
		},
		{
			name: "MQTT enabled but no host",
			yaml: `
solarController:
  httpPort: 8080
  mqtt:
    enabled: true
    host: ""
    topicPrefix: solar/metrics
  epever:
    enabled: false
`,
			wantErr: true,
			errMsg:  "MQTT host is required",
		},
		{
			name: "MQTT enabled but no topic prefix",
			yaml: `
solarController:
  httpPort: 8080
  mqtt:
    enabled: true
    host: mqtt://localhost:1883
    topicPrefix: ""
  epever:
    enabled: false
`,
			wantErr: true,
			errMsg:  "MQTT topic prefix is required",
		},
		{
			name: "MQTT disabled - no validation errors",
			yaml: `
solarController:
  httpPort: 8080
  mqtt:
    enabled: false
  epever:
    enabled: false
`,
			wantErr: false,
		},
		{
			name: "epever enabled but no serial port",
			yaml: `
solarController:
  httpPort: 8080
  epever:
    enabled: true
    serialPort: ""
    publishPeriod: 60
`,
			wantErr: true,
			errMsg:  "epever serial port is required",
		},
		{
			name: "epever enabled but invalid publish period",
			yaml: `
solarController:
  httpPort: 8080
  epever:
    enabled: true
    serialPort: /dev/ttyUSB0
    publishPeriod: 0
`,
			wantErr: true,
			errMsg:  "epever publish period must be positive",
		},
		{
			name: "epever enabled but negative publish period",
			yaml: `
solarController:
  httpPort: 8080
  epever:
    enabled: true
    serialPort: /dev/ttyUSB0
    publishPeriod: -10
`,
			wantErr: true,
			errMsg:  "epever publish period must be positive",
		},
		{
			name: "MQTT configuration valid with all fields",
			yaml: `
solarController:
  httpPort: 8080
  mqtt:
    enabled: true
    host: mqtt://broker.example.com:1883
    username: solaruser
    password: secretpassword
    topicPrefix: home/solar
  epever:
    enabled: false
`,
			wantErr: false,
			check: func(t *testing.T, c Config) {
				if !c.SolarController.Mqtt.Enabled {
					t.Error("MQTT should be enabled")
				}
				if c.SolarController.Mqtt.Username != "solaruser" {
					t.Errorf("MQTT Username = %s, want solaruser", c.SolarController.Mqtt.Username)
				}
				if c.SolarController.Mqtt.Password != "secretpassword" {
					t.Errorf("MQTT Password = %s, want secretpassword", c.SolarController.Mqtt.Password)
				}
			},
		},
		{
			name:    "empty configuration",
			yaml:    ``,
			wantErr: true,
			errMsg:  "invalid HTTP port",
		},
		{
			name: "missing solarController section",
			yaml: `
someOtherConfig:
  value: 123
`,
			wantErr: true,
			errMsg:  "invalid HTTP port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := Load([]byte(tt.yaml))

			if tt.wantErr {
				if err == nil {
					t.Errorf("Load() expected error containing '%s', got nil", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Load() error = '%v', want error containing '%s'", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Load() unexpected error: %v", err)
					return
				}
				if tt.check != nil {
					tt.check(t, config)
				}
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			config: Config{
				SolarController: SolarControllerConfiguration{
					HTTPPort: 8080,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid HTTP port - boundary test at 0",
			config: Config{
				SolarController: SolarControllerConfiguration{
					HTTPPort: 0,
				},
			},
			wantErr: true,
			errMsg:  "invalid HTTP port",
		},
		{
			name: "invalid HTTP port - boundary test at 65536",
			config: Config{
				SolarController: SolarControllerConfiguration{
					HTTPPort: 65536,
				},
			},
			wantErr: true,
			errMsg:  "invalid HTTP port",
		},
		{
			name: "valid HTTP port - boundary test at 1",
			config: Config{
				SolarController: SolarControllerConfiguration{
					HTTPPort: 1,
				},
			},
			wantErr: false,
		},
		{
			name: "valid HTTP port - boundary test at 65535",
			config: Config{
				SolarController: SolarControllerConfiguration{
					HTTPPort: 65535,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing '%s', got nil", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = '%v', want error containing '%s'", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}
