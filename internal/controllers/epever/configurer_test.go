package epever

import (
	"testing"
)

func TestValidateVoltageParameters(t *testing.T) {
	tests := []struct {
		name        string
		config      ControllerConfig
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			config: ControllerConfig{
				OverVoltDisconnectVoltage:     15.5,
				ChargingLimitVoltage:          15.0,
				EqualizationVoltage:           14.6,
				BoostVoltage:                  14.4,
				FloatVoltage:                  13.8,
				BoostReconnectChargingVoltage: 13.2,
				UnderVoltReconnectVoltage:     12.6,
				UnderVoltWarningVoltage:       12.0,
				LowVoltDisconnectVoltage:      11.1,
				DischargingLimitVoltage:       10.8,
				OverVoltReconnectVoltage:      15.0,
				LowVoltReconnectVoltage:       11.5,
			},
			shouldError: false,
		},
		{
			name: "invalid charging voltage chain - boost > equalization",
			config: ControllerConfig{
				OverVoltDisconnectVoltage:     15.5,
				ChargingLimitVoltage:          15.0,
				EqualizationVoltage:           14.0, // Lower than boost
				BoostVoltage:                  14.4, // Higher than equalization
				FloatVoltage:                  13.8,
				BoostReconnectChargingVoltage: 13.2,
				UnderVoltReconnectVoltage:     12.6,
				UnderVoltWarningVoltage:       12.0,
				LowVoltDisconnectVoltage:      11.1,
				DischargingLimitVoltage:       10.8,
				OverVoltReconnectVoltage:      15.0,
				LowVoltReconnectVoltage:       11.5,
			},
			shouldError: true,
			errorMsg:    "charging voltage chain violated",
		},
		{
			name: "invalid discharging voltage chain",
			config: ControllerConfig{
				OverVoltDisconnectVoltage:     15.5,
				ChargingLimitVoltage:          15.0,
				EqualizationVoltage:           14.6,
				BoostVoltage:                  14.4,
				FloatVoltage:                  13.8,
				BoostReconnectChargingVoltage: 13.2,
				UnderVoltReconnectVoltage:     12.6,
				UnderVoltWarningVoltage:       12.0,
				LowVoltDisconnectVoltage:      10.5, // Lower than discharging limit
				DischargingLimitVoltage:       10.8, // Higher than low volt disconnect
				OverVoltReconnectVoltage:      15.0,
				LowVoltReconnectVoltage:       11.5,
			},
			shouldError: true,
			errorMsg:    "discharging voltage chain violated",
		},
		{
			name: "invalid over voltage pair",
			config: ControllerConfig{
				OverVoltDisconnectVoltage:     15.0, // Lower than reconnect
				ChargingLimitVoltage:          15.0,
				EqualizationVoltage:           14.6,
				BoostVoltage:                  14.4,
				FloatVoltage:                  13.8,
				BoostReconnectChargingVoltage: 13.2,
				UnderVoltReconnectVoltage:     12.6,
				UnderVoltWarningVoltage:       12.0,
				LowVoltDisconnectVoltage:      11.1,
				DischargingLimitVoltage:       10.8,
				OverVoltReconnectVoltage:      15.5, // Higher than disconnect
				LowVoltReconnectVoltage:       11.5,
			},
			shouldError: true,
			errorMsg:    "over voltage pair violated",
		},
		{
			name: "invalid low voltage pair",
			config: ControllerConfig{
				OverVoltDisconnectVoltage:     15.5,
				ChargingLimitVoltage:          15.0,
				EqualizationVoltage:           14.6,
				BoostVoltage:                  14.4,
				FloatVoltage:                  13.8,
				BoostReconnectChargingVoltage: 13.2,
				UnderVoltReconnectVoltage:     12.6,
				UnderVoltWarningVoltage:       12.0,
				LowVoltDisconnectVoltage:      11.1,
				DischargingLimitVoltage:       10.8,
				OverVoltReconnectVoltage:      15.0,
				LowVoltReconnectVoltage:       11.0, // Lower than disconnect
			},
			shouldError: true,
			errorMsg:    "low voltage pair violated",
		},
		{
			name: "invalid charging voltage chain - float <= boostReconnect",
			config: ControllerConfig{
				OverVoltDisconnectVoltage:     15.5,
				ChargingLimitVoltage:          15.0,
				EqualizationVoltage:           14.6,
				BoostVoltage:                  14.4,
				FloatVoltage:                  13.2, // Equal to boost reconnect
				BoostReconnectChargingVoltage: 13.2,
				UnderVoltReconnectVoltage:     12.6,
				UnderVoltWarningVoltage:       12.0,
				LowVoltDisconnectVoltage:      11.1,
				DischargingLimitVoltage:       10.8,
				OverVoltReconnectVoltage:      15.0,
				LowVoltReconnectVoltage:       11.5,
			},
			shouldError: true,
			errorMsg:    "charging voltage chain violated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVoltageParameters(&tt.config)
			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errorMsg)
				} else if err.Error() == "" {
					t.Errorf("expected error message, got empty string")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}
		})
	}
}
