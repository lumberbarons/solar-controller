package epever

import (
	"context"
	"encoding/binary"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// Modbus holding register addresses (read-write configuration values)
const (
	regBatteryType               = 0x9000
	regBatteryCapacity           = 0x9001
	regTempCompCoefficient       = 0x9002
	regOverVoltDisconnect        = 0x9003
	regChargingLimitVoltage      = 0x9004
	regOverVoltReconnect         = 0x9005
	regEqualizationVoltage       = 0x9006
	regBoostVoltage              = 0x9007
	regFloatVoltage              = 0x9008
	regBoostReconnectVoltage     = 0x9009
	regLowVoltReconnect          = 0x900A
	regUnderVoltRecover          = 0x900B
	regUnderVoltWarning          = 0x900C
	regLowVoltDisconnect         = 0x900D
	regDischargingLimitVoltage   = 0x900E
	regRealTimeClock             = 0x9013
	regEqualizationChargingCycle = 0x9016
	regBatteryTempUpperLimit     = 0x9017
	regBatteryTempLowerLimit     = 0x9018
	regControllerTempUpperLimit  = 0x9019
	regControllerTempLowerLimit  = 0x901A
	regEqualizationChargingTime  = 0x906B
	regBoostChargingTime         = 0x906C
)

// Battery type constants
const (
	batteryTypeUserDefined = 0
)

// Conversion factor for voltage values (stored as centivolt)
const voltageDivisor = 100.0

// maxPatchBodyBytes caps config PATCH request bodies before they are
// buffered by JSON binding; the payloads these endpoints accept are at
// most a few hundred bytes.
const maxPatchBodyBytes = 8 << 10

// bindJSONBounded binds a JSON request body while enforcing maxPatchBodyBytes,
// so oversized bodies are rejected instead of exhausting memory.
func bindJSONBounded(c *gin.Context, obj any) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxPatchBodyBytes)
	return c.BindJSON(obj)
}

// Absolute bounds for values written to the controller. The voltage range is
// generous enough for 12/24/48V battery systems while rejecting values that
// would wrap the uint16 centivolt registers (overflow at 655.35 V).
const (
	minVoltageSetting          = 5.0
	maxVoltageSetting          = 70.0
	minBatteryCapacityAh       = 1
	maxBatteryCapacityAh       = 10000
	minTempCompCoefficient     = 0.0
	maxTempCompCoefficient     = 10.0
	maxChargingDurationMinutes = 600
	maxEqualizationCycleDays   = 365
	minRTCYear                 = 2000
	maxRTCYear                 = 2255 // year is stored as a single byte offset from 2000
)

// validateVoltageBounds rejects voltages outside the absolute allowed range.
func validateVoltageBounds(name string, volts float32) error {
	if volts < minVoltageSetting || volts > maxVoltageSetting {
		return fmt.Errorf("%s (%.2f) out of range [%.1f, %.1f] volts", name, volts, minVoltageSetting, maxVoltageSetting)
	}
	return nil
}

type Configurer struct {
	modbusClient        ModbusClient
	prometheusCollector MetricsCollector
	cache               *cachedConfig
	cacheMutex          sync.RWMutex
	cacheTTL            time.Duration
}

func NewConfigurer(client ModbusClient, prometheusCollector MetricsCollector) *Configurer {
	return &Configurer{
		modbusClient:        client,
		prometheusCollector: prometheusCollector,
		cacheTTL:            10 * time.Minute,
	}
}

type ControllerConfig struct {
	Time                          string  `json:"time"`
	BatteryType                   string  `json:"batteryType"`
	BatteryCapacity               uint16  `json:"batteryCapacity"`
	TempCompCoefficient           float32 `json:"tempCompCoefficient"`
	BoostDuration                 uint16  `json:"boostDuration"`
	EqualizationCycle             uint16  `json:"equalizationCycle"`
	EqualizationDuration          uint16  `json:"equalizationDuration"`
	BoostVoltage                  float32 `json:"boostVoltage"`
	BoostReconnectChargingVoltage float32 `json:"boostReconnectChargingVoltage"`
	FloatVoltage                  float32 `json:"floatVoltage"`
	EqualizationVoltage           float32 `json:"equalizationVoltage"`
	ChargingLimitVoltage          float32 `json:"chargingLimitVoltage"`
	OverVoltDisconnectVoltage     float32 `json:"overVoltDisconnectVoltage"`
	OverVoltReconnectVoltage      float32 `json:"overVoltReconnectVoltage"`
	LowVoltDisconnectVoltage      float32 `json:"lowVoltDisconnectVoltage"`
	LowVoltReconnectVoltage       float32 `json:"lowVoltReconnectVoltage"`
	UnderVoltWarningVoltage       float32 `json:"underVoltWarningVoltage"`
	UnderVoltReconnectVoltage     float32 `json:"underVoltWarningReconnectVoltage"`
	DischargingLimitVoltage       float32 `json:"dischargingLimitVoltage"`
	BatteryTempUpperLimit         float32 `json:"batteryTempUpperLimit"`
	BatteryTempLowerLimit         float32 `json:"batteryTempLowerLimit"`
	ControllerTempUpperLimit      float32 `json:"controllerTempUpperLimit"`
	ControllerTempLowerLimit      float32 `json:"controllerTempLowerLimit"`
}

func (sc *Configurer) ConfigGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		config, err := sc.getConfig(c.Request.Context())
		if err != nil {
			log.Warn("Failed to get config", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, config)
	}
}

func (sc *Configurer) getFloatValue(data []byte, index int) float32 {
	offset := index * 2
	if len(data) < offset+2 {
		log.Warnf("getFloatValue: insufficient data at index %d (need %d bytes, have %d)", index, offset+2, len(data))
		return 0
	}
	return float32(binary.BigEndian.Uint16(data[offset:offset+2])) / voltageDivisor
}

func (sc *Configurer) getConfig(ctx context.Context) (ControllerConfig, error) {
	startTime := time.Now()
	log.Debug("Starting config read from device")

	// Read battery type, capacity, and temp compensation coefficient
	data, err := sc.modbusClient.ReadHoldingRegisters(ctx, regBatteryType, 3)
	if err != nil {
		sc.prometheusCollector.IncrementRegisterFailure(regBatteryType, "holding")
		return ControllerConfig{}, fmt.Errorf("failed to read battery config (0x%X): %w", regBatteryType, err)
	}
	time.Sleep(75 * time.Millisecond) // Allow device to recover before next read

	batteryType := binary.BigEndian.Uint16(data[0:2])
	batteryCapacity := binary.BigEndian.Uint16(data[2:4])
	tempCompCoefficient := sc.getFloatValue(data, 2)

	// Read time
	data, err = sc.modbusClient.ReadHoldingRegisters(ctx, regRealTimeClock, 3)
	if err != nil {
		sc.prometheusCollector.IncrementRegisterFailure(regRealTimeClock, "holding")
		return ControllerConfig{}, fmt.Errorf("failed to read time (0x%X): %w", regRealTimeClock, err)
	}
	time.Sleep(75 * time.Millisecond) // Allow device to recover before next read
	if len(data) < 6 {
		return ControllerConfig{}, fmt.Errorf("insufficient time data: expected 6 bytes, got %d", len(data))
	}

	year := int(data[4]) + 2000
	timeStr := fmt.Sprintf("%d-%d-%d %02d:%02d:%02d",
		data[2], data[5], year, data[3], data[0], data[1])

	// Read voltage parameters (largest read - 12 registers)
	data, err = sc.modbusClient.ReadHoldingRegisters(ctx, regOverVoltDisconnect, 12)
	if err != nil {
		sc.prometheusCollector.IncrementRegisterFailure(regOverVoltDisconnect, "holding")
		return ControllerConfig{}, fmt.Errorf("failed to read voltage parameters (0x%X): %w", regOverVoltDisconnect, err)
	}
	time.Sleep(100 * time.Millisecond) // Extra delay after large read
	overVoltDisconnectVoltage := sc.getFloatValue(data, 0)
	chargingLimitVoltage := sc.getFloatValue(data, 1)
	overVoltReconnectVoltage := sc.getFloatValue(data, 2)
	equalizationVoltage := sc.getFloatValue(data, 3)
	boostVoltage := sc.getFloatValue(data, 4)
	floatVoltage := sc.getFloatValue(data, 5)
	boostReconnectVoltage := sc.getFloatValue(data, 6)
	lowVoltageReconnect := sc.getFloatValue(data, 7)
	underVoltageRecover := sc.getFloatValue(data, 8)
	underVoltageWarning := sc.getFloatValue(data, 9)
	lowVoltageDisconnect := sc.getFloatValue(data, 10)
	dischargingLimitVoltage := sc.getFloatValue(data, 11)

	// Read equalization cycle
	data, err = sc.modbusClient.ReadHoldingRegisters(ctx, regEqualizationChargingCycle, 1)
	if err != nil {
		sc.prometheusCollector.IncrementRegisterFailure(regEqualizationChargingCycle, "holding")
		return ControllerConfig{}, fmt.Errorf("failed to read equalization cycle (0x%X): %w", regEqualizationChargingCycle, err)
	}
	time.Sleep(75 * time.Millisecond) // Allow device to recover before next read
	if len(data) < 2 {
		return ControllerConfig{}, fmt.Errorf("insufficient equalization cycle data: expected 2 bytes, got %d", len(data))
	}
	equalizationCycle := binary.BigEndian.Uint16(data[0:2])

	// Read durations
	data, err = sc.modbusClient.ReadHoldingRegisters(ctx, regEqualizationChargingTime, 2)
	if err != nil {
		sc.prometheusCollector.IncrementRegisterFailure(regEqualizationChargingTime, "holding")
		return ControllerConfig{}, fmt.Errorf("failed to read durations (0x%X): %w", regEqualizationChargingTime, err)
	}
	time.Sleep(75 * time.Millisecond) // Allow device to recover before next read
	if len(data) < 4 {
		return ControllerConfig{}, fmt.Errorf("insufficient duration data: expected 4 bytes, got %d", len(data))
	}
	equalizationDuration := binary.BigEndian.Uint16(data[0:2])
	boostDuration := binary.BigEndian.Uint16(data[2:4])

	// Read temperature limits
	data, err = sc.modbusClient.ReadHoldingRegisters(ctx, regBatteryTempUpperLimit, 4)
	if err != nil {
		sc.prometheusCollector.IncrementRegisterFailure(regBatteryTempUpperLimit, "holding")
		return ControllerConfig{}, fmt.Errorf("failed to read temperature limits (0x%X): %w", regBatteryTempUpperLimit, err)
	}
	// No delay needed after final read
	if len(data) < 8 {
		return ControllerConfig{}, fmt.Errorf("insufficient temperature data: expected 8 bytes, got %d", len(data))
	}
	batteryTempUpperLimit := float32(int16(binary.BigEndian.Uint16(data[0:2]))) / voltageDivisor
	batteryTempLowerLimit := float32(int16(binary.BigEndian.Uint16(data[2:4]))) / voltageDivisor
	controllerTempUpperLimit := float32(int16(binary.BigEndian.Uint16(data[4:6]))) / voltageDivisor
	controllerTempLowerLimit := float32(int16(binary.BigEndian.Uint16(data[6:8]))) / voltageDivisor

	elapsed := time.Since(startTime)
	log.Debugf("Config read completed in %v", elapsed)

	return ControllerConfig{
		Time:                          timeStr,
		BatteryType:                   batteryTypeToString(batteryType),
		BatteryCapacity:               batteryCapacity,
		TempCompCoefficient:           tempCompCoefficient,
		BoostDuration:                 boostDuration,
		EqualizationDuration:          equalizationDuration,
		EqualizationCycle:             equalizationCycle,
		EqualizationVoltage:           equalizationVoltage,
		BoostVoltage:                  boostVoltage,
		FloatVoltage:                  floatVoltage,
		BoostReconnectChargingVoltage: boostReconnectVoltage,
		OverVoltDisconnectVoltage:     overVoltDisconnectVoltage,
		ChargingLimitVoltage:          chargingLimitVoltage,
		OverVoltReconnectVoltage:      overVoltReconnectVoltage,
		LowVoltReconnectVoltage:       lowVoltageReconnect,
		UnderVoltReconnectVoltage:     underVoltageRecover,
		UnderVoltWarningVoltage:       underVoltageWarning,
		LowVoltDisconnectVoltage:      lowVoltageDisconnect,
		DischargingLimitVoltage:       dischargingLimitVoltage,
		BatteryTempUpperLimit:         batteryTempUpperLimit,
		BatteryTempLowerLimit:         batteryTempLowerLimit,
		ControllerTempUpperLimit:      controllerTempUpperLimit,
		ControllerTempLowerLimit:      controllerTempLowerLimit,
	}, nil
}

func (sc *Configurer) writeSingle(c *gin.Context, address, value uint16, description string) error {
	log.Info(fmt.Sprintf("Setting %v of %v to controller", description, value))
	// Convert uint16 value to byte array (BigEndian) for WriteMultipleRegisters
	// Function code 0x10 is required per Epever documentation for all Holding Register writes
	bytes := make([]byte, 2)
	binary.BigEndian.PutUint16(bytes, value)
	_, err := sc.modbusClient.WriteMultipleRegisters(c.Request.Context(), address, 1, bytes)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to write %v of %v to controller", description, value)
		log.Warn(errorMessage, err.Error())
		if sc.prometheusCollector != nil {
			sc.prometheusCollector.IncrementWriteFailures()
		}
		return fmt.Errorf("%s: %w", errorMessage, err)
	}
	// Allow device time to commit write to EEPROM before next operation
	time.Sleep(150 * time.Millisecond)
	return nil
}

// writeVoltageParametersBlock writes all 12 voltage parameter registers (0x9003-0x900E) in a single operation.
// For any voltage parameters not provided in the config, it uses values from the cached config.
func (sc *Configurer) writeVoltageParametersBlock(c *gin.Context, config *ControllerConfig) error {
	ctx := c.Request.Context()

	// Get current config from cache to fill in any missing values
	cachedConfig, err := sc.getCachedConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get cached config for voltage parameters: %w", err)
	}

	// Use provided values, fall back to cached values if not provided
	overVoltDisconnect := config.OverVoltDisconnectVoltage
	if overVoltDisconnect == 0 {
		overVoltDisconnect = cachedConfig.OverVoltDisconnectVoltage
	}
	chargingLimit := config.ChargingLimitVoltage
	if chargingLimit == 0 {
		chargingLimit = cachedConfig.ChargingLimitVoltage
	}
	overVoltReconnect := config.OverVoltReconnectVoltage
	if overVoltReconnect == 0 {
		overVoltReconnect = cachedConfig.OverVoltReconnectVoltage
	}
	equalization := config.EqualizationVoltage
	if equalization == 0 {
		equalization = cachedConfig.EqualizationVoltage
	}
	boost := config.BoostVoltage
	if boost == 0 {
		boost = cachedConfig.BoostVoltage
	}
	floatVolt := config.FloatVoltage
	if floatVolt == 0 {
		floatVolt = cachedConfig.FloatVoltage
	}
	boostReconnect := config.BoostReconnectChargingVoltage
	if boostReconnect == 0 {
		boostReconnect = cachedConfig.BoostReconnectChargingVoltage
	}
	lowVoltReconnect := config.LowVoltReconnectVoltage
	if lowVoltReconnect == 0 {
		lowVoltReconnect = cachedConfig.LowVoltReconnectVoltage
	}
	underVoltRecover := config.UnderVoltReconnectVoltage
	if underVoltRecover == 0 {
		underVoltRecover = cachedConfig.UnderVoltReconnectVoltage
	}
	underVoltWarning := config.UnderVoltWarningVoltage
	if underVoltWarning == 0 {
		underVoltWarning = cachedConfig.UnderVoltWarningVoltage
	}
	lowVoltDisconnect := config.LowVoltDisconnectVoltage
	if lowVoltDisconnect == 0 {
		lowVoltDisconnect = cachedConfig.LowVoltDisconnectVoltage
	}
	dischargingLimit := config.DischargingLimitVoltage
	if dischargingLimit == 0 {
		dischargingLimit = cachedConfig.DischargingLimitVoltage
	}

	// Registers 0x9003-0x900E in ascending address order
	merged := []struct {
		name  string
		volts float32
	}{
		{"overVoltDisconnectVoltage", overVoltDisconnect},
		{"chargingLimitVoltage", chargingLimit},
		{"overVoltReconnectVoltage", overVoltReconnect},
		{"equalizationVoltage", equalization},
		{"boostVoltage", boost},
		{"floatVoltage", floatVolt},
		{"boostReconnectChargingVoltage", boostReconnect},
		{"lowVoltReconnectVoltage", lowVoltReconnect},
		{"underVoltWarningReconnectVoltage", underVoltRecover},
		{"underVoltWarningVoltage", underVoltWarning},
		{"lowVoltDisconnectVoltage", lowVoltDisconnect},
		{"dischargingLimitVoltage", dischargingLimit},
	}

	// Convert all voltage values to uint16 (multiply by 100 for device
	// format), rejecting anything that would overflow the register
	values := make([]uint16, len(merged))
	for i, m := range merged {
		if err := validateVoltageBounds(m.name, m.volts); err != nil {
			return err
		}
		values[i] = uint16(m.volts * voltageDivisor)
	}

	// Build byte array for all 12 registers (24 bytes)
	bytes := make([]byte, 24)
	for i, value := range values {
		binary.BigEndian.PutUint16(bytes[i*2:], value)
	}

	log.Info("Writing voltage parameters block (0x9003-0x900E) to controller")
	_, err = sc.modbusClient.WriteMultipleRegisters(ctx, regOverVoltDisconnect, 12, bytes)
	if err != nil {
		log.Warn("Failed to write voltage parameters block", err.Error())
		if sc.prometheusCollector != nil {
			sc.prometheusCollector.IncrementWriteFailures()
		}
		return fmt.Errorf("failed to write voltage parameters block: %w", err)
	}

	// Allow device time to commit all writes to EEPROM
	time.Sleep(500 * time.Millisecond)
	return nil
}

func (sc *Configurer) ConfigPatch() gin.HandlerFunc {
	return func(c *gin.Context) {
		var config ControllerConfig
		err := bindJSONBounded(c, &config)
		if err != nil {
			log.Warn("Config patch bad json request", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate before any write so an invalid request changes nothing
		var requestedType uint16
		typeRequested := config.BatteryType != ""
		if typeRequested {
			requestedType, err = parseBatteryType(config.BatteryType)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
		if config.EqualizationCycle > maxEqualizationCycleDays {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("equalizationCycle (%d) out of range [0, %d] days", config.EqualizationCycle, maxEqualizationCycleDays)})
			return
		}
		if config.EqualizationDuration > maxChargingDurationMinutes {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("equalizationDuration (%d) out of range [0, %d] minutes", config.EqualizationDuration, maxChargingDurationMinutes)})
			return
		}
		if config.BoostDuration > maxChargingDurationMinutes {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("boostDuration (%d) out of range [0, %d] minutes", config.BoostDuration, maxChargingDurationMinutes)})
			return
		}

		// Determine the effective battery type without writing anything yet
		userDefined := false
		if typeRequested {
			userDefined = requestedType == batteryTypeUserDefined
		} else {
			data, err := sc.modbusClient.ReadHoldingRegisters(c.Request.Context(), regBatteryType, 1)
			if err != nil {
				log.Warn("Failed to read battery type", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read battery type"})
				return
			}
			userDefined = binary.BigEndian.Uint16(data[0:2]) == batteryTypeUserDefined
		}

		var proposedConfig ControllerConfig
		voltageParamsPresent := false
		if userDefined {
			// Get current configuration for validation
			currentConfig, err := sc.getCachedConfig(c.Request.Context())
			if err != nil {
				log.Warn("Failed to read current config for validation", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read current configuration"})
				return
			}

			// Create a copy of the current config to apply proposed changes
			proposedConfig = *currentConfig

			// Apply requested changes to proposed config for validation
			if config.ChargingLimitVoltage > 0 {
				proposedConfig.ChargingLimitVoltage = config.ChargingLimitVoltage
			}

			if config.EqualizationVoltage > 0 {
				proposedConfig.EqualizationVoltage = config.EqualizationVoltage
			}

			if config.BoostVoltage > 0 {
				proposedConfig.BoostVoltage = config.BoostVoltage
			}

			if config.FloatVoltage > 0 {
				proposedConfig.FloatVoltage = config.FloatVoltage
			}

			if config.BoostReconnectChargingVoltage > 0 {
				proposedConfig.BoostReconnectChargingVoltage = config.BoostReconnectChargingVoltage
			}

			// Validate the proposed configuration
			if err := validateVoltageParameters(&proposedConfig); err != nil {
				log.Warn("Voltage parameter validation failed", err)
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// Check if any voltage parameters are present
			voltageParamsPresent = config.ChargingLimitVoltage > 0 ||
				config.EqualizationVoltage > 0 ||
				config.BoostVoltage > 0 ||
				config.FloatVoltage > 0 ||
				config.BoostReconnectChargingVoltage > 0
		}

		// All validation passed: perform the writes
		if typeRequested {
			if err := sc.writeSingle(c, regBatteryType, requestedType, "battery type"); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			// Invalidate cache when battery type is changed
			sc.invalidateCache()
		}

		if userDefined {
			// If voltage parameters are present, write the entire block
			if voltageParamsPresent {
				if err := sc.writeVoltageParametersBlock(c, &proposedConfig); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}

			// Write non-voltage parameters individually
			if config.EqualizationCycle > 0 {
				if err := sc.writeSingle(c, regEqualizationChargingCycle, config.EqualizationCycle, "equalization cycle"); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
			}

			if config.EqualizationDuration > 0 {
				if err := sc.writeSingle(c, regEqualizationChargingTime, config.EqualizationDuration, "equalization duration"); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
			}

			if config.BoostDuration > 0 {
				if err := sc.writeSingle(c, regBoostChargingTime, config.BoostDuration, "boost duration"); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
			}

			// Invalidate cache after writing charging parameters
			sc.invalidateCache()
		}

		newConfig, err := sc.getCachedConfig(c.Request.Context())
		if err != nil {
			log.Warn("Failed to retrieve updated config after write", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Configuration updated but failed to read back"})
			return
		}
		c.JSON(http.StatusOK, newConfig)
	}
}

// validateVoltageParameters validates voltage magnitudes and relationships
// according to modbus register documentation. Returns an error if any value
// falls outside the absolute bounds or violates a voltage relationship rule.
func validateVoltageParameters(config *ControllerConfig) error {
	// Absolute bounds: every writable voltage register must be in range
	voltages := []struct {
		name  string
		volts float32
	}{
		{"overVoltDisconnectVoltage", config.OverVoltDisconnectVoltage},
		{"chargingLimitVoltage", config.ChargingLimitVoltage},
		{"overVoltReconnectVoltage", config.OverVoltReconnectVoltage},
		{"equalizationVoltage", config.EqualizationVoltage},
		{"boostVoltage", config.BoostVoltage},
		{"floatVoltage", config.FloatVoltage},
		{"boostReconnectChargingVoltage", config.BoostReconnectChargingVoltage},
		{"lowVoltReconnectVoltage", config.LowVoltReconnectVoltage},
		{"underVoltWarningReconnectVoltage", config.UnderVoltReconnectVoltage},
		{"underVoltWarningVoltage", config.UnderVoltWarningVoltage},
		{"lowVoltDisconnectVoltage", config.LowVoltDisconnectVoltage},
		{"dischargingLimitVoltage", config.DischargingLimitVoltage},
	}
	for _, v := range voltages {
		if err := validateVoltageBounds(v.name, v.volts); err != nil {
			return err
		}
	}

	// Rule 1: Charging voltage chain
	// Over voltage disconnect > Charge limit > Equalize charging > Boost charging > Float charging > Boost reconnect charging
	if !(config.OverVoltDisconnectVoltage > config.ChargingLimitVoltage &&
		config.ChargingLimitVoltage > config.EqualizationVoltage &&
		config.EqualizationVoltage > config.BoostVoltage &&
		config.BoostVoltage > config.FloatVoltage &&
		config.FloatVoltage > config.BoostReconnectChargingVoltage) {
		return fmt.Errorf("charging voltage chain violated: overVoltDisconnect (%.2f) > chargingLimit (%.2f) > equalization (%.2f) > boost (%.2f) > float (%.2f) > boostReconnect (%.2f)",
			config.OverVoltDisconnectVoltage, config.ChargingLimitVoltage, config.EqualizationVoltage,
			config.BoostVoltage, config.FloatVoltage, config.BoostReconnectChargingVoltage)
	}

	// Rule 2: Discharging voltage chain
	// Under voltage warning recover > Under voltage warning > Low voltage disconnect > Discharging limit
	if !(config.UnderVoltReconnectVoltage > config.UnderVoltWarningVoltage &&
		config.UnderVoltWarningVoltage > config.LowVoltDisconnectVoltage &&
		config.LowVoltDisconnectVoltage > config.DischargingLimitVoltage) {
		return fmt.Errorf("discharging voltage chain violated: underVoltReconnect (%.2f) > underVoltWarning (%.2f) > lowVoltDisconnect (%.2f) > dischargingLimit (%.2f)",
			config.UnderVoltReconnectVoltage, config.UnderVoltWarningVoltage,
			config.LowVoltDisconnectVoltage, config.DischargingLimitVoltage)
	}

	// Rule 3: Over voltage pair
	// Over voltage disconnect > Over voltage reconnect
	if !(config.OverVoltDisconnectVoltage > config.OverVoltReconnectVoltage) {
		return fmt.Errorf("over voltage pair violated: overVoltDisconnect (%.2f) > overVoltReconnect (%.2f)",
			config.OverVoltDisconnectVoltage, config.OverVoltReconnectVoltage)
	}

	// Rule 4: Low voltage pair
	// Low voltage reconnect > Low voltage disconnect
	if !(config.LowVoltReconnectVoltage > config.LowVoltDisconnectVoltage) {
		return fmt.Errorf("low voltage pair violated: lowVoltReconnect (%.2f) > lowVoltDisconnect (%.2f)",
			config.LowVoltReconnectVoltage, config.LowVoltDisconnectVoltage)
	}

	return nil
}
