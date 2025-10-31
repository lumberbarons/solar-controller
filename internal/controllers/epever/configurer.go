package epever

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lumberbarons/solar-controller/internal/controllers"
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
	batteryTypeUserDefined = 4
)

// Conversion factor for voltage values (stored as centivolt)
const voltageDivisor = 100.0

type cachedConfig struct {
	config    *ControllerConfig
	timestamp time.Time
}

type Configurer struct {
	modbusClient        controllers.ModbusClient
	prometheusCollector controllers.MetricsCollector
	cache               *cachedConfig
	cacheMutex          sync.RWMutex
	cacheTTL            time.Duration
}

func NewConfigurer(client controllers.ModbusClient, prometheusCollector controllers.MetricsCollector) *Configurer {
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

// BatteryProfile contains battery identity settings
type BatteryProfile struct {
	BatteryType     *string `json:"batteryType,omitempty"`
	BatteryCapacity *uint16 `json:"batteryCapacity,omitempty"`
}

// ChargingParameters contains all charging algorithm settings
type ChargingParameters struct {
	BoostDuration                 *uint16  `json:"boostDuration,omitempty"`
	EqualizationCycle             *uint16  `json:"equalizationCycle,omitempty"`
	EqualizationDuration          *uint16  `json:"equalizationDuration,omitempty"`
	BoostVoltage                  *float32 `json:"boostVoltage,omitempty"`
	BoostReconnectChargingVoltage *float32 `json:"boostReconnectChargingVoltage,omitempty"`
	FloatVoltage                  *float32 `json:"floatVoltage,omitempty"`
	EqualizationVoltage           *float32 `json:"equalizationVoltage,omitempty"`
	ChargingLimitVoltage          *float32 `json:"chargingLimitVoltage,omitempty"`
	OverVoltDisconnectVoltage     *float32 `json:"overVoltDisconnectVoltage,omitempty"`
	OverVoltReconnectVoltage      *float32 `json:"overVoltReconnectVoltage,omitempty"`
	LowVoltDisconnectVoltage      *float32 `json:"lowVoltDisconnectVoltage,omitempty"`
	LowVoltReconnectVoltage       *float32 `json:"lowVoltReconnectVoltage,omitempty"`
	UnderVoltWarningVoltage       *float32 `json:"underVoltWarningVoltage,omitempty"`
	UnderVoltReconnectVoltage     *float32 `json:"underVoltWarningReconnectVoltage,omitempty"`
	DischargingLimitVoltage       *float32 `json:"dischargingLimitVoltage,omitempty"`
	BatteryTempUpperLimit         *float32 `json:"batteryTempUpperLimit,omitempty"`
	BatteryTempLowerLimit         *float32 `json:"batteryTempLowerLimit,omitempty"`
	ControllerTempUpperLimit      *float32 `json:"controllerTempUpperLimit,omitempty"`
	ControllerTempLowerLimit      *float32 `json:"controllerTempLowerLimit,omitempty"`
}

type TimeConfig struct {
	Time time.Time `json:"time"`
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

// getCachedConfig returns the cached config if valid, otherwise fetches from device
func (sc *Configurer) getCachedConfig(ctx context.Context) (*ControllerConfig, error) {
	sc.cacheMutex.RLock()
	if sc.cache != nil && time.Since(sc.cache.timestamp) < sc.cacheTTL {
		// Return a copy to prevent external modification of cached data
		configCopy := *sc.cache.config
		sc.cacheMutex.RUnlock()
		log.Trace("Using cached config")
		return &configCopy, nil
	}
	sc.cacheMutex.RUnlock()

	// Cache miss or expired - fetch from device
	config, err := sc.getConfig(ctx)
	if err != nil {
		return nil, err
	}

	// Update cache with a copy
	configCopy := config
	sc.cacheMutex.Lock()
	sc.cache = &cachedConfig{
		config:    &configCopy,
		timestamp: time.Now(),
	}
	sc.cacheMutex.Unlock()

	log.Trace("Fetched and cached config from device")
	return &config, nil
}

// invalidateCache clears the cache, forcing the next read to fetch from device
func (sc *Configurer) invalidateCache() {
	sc.cacheMutex.Lock()
	sc.cache = nil
	sc.cacheMutex.Unlock()
	log.Trace("Config cache invalidated")
}

func (sc *Configurer) getConfig(ctx context.Context) (ControllerConfig, error) {
	// Read battery type, capacity, and temp compensation coefficient
	data, err := sc.modbusClient.ReadHoldingRegisters(ctx, regBatteryType, 3)
	if err != nil {
		return ControllerConfig{}, fmt.Errorf("failed to read battery config (0x%X): %w", regBatteryType, err)
	}

	batteryType := binary.BigEndian.Uint16(data[0:2])
	batteryCapacity := binary.BigEndian.Uint16(data[2:4])
	tempCompCoefficient := sc.getFloatValue(data, 2)

	// Read time
	data, err = sc.modbusClient.ReadHoldingRegisters(ctx, regRealTimeClock, 3)
	if err != nil {
		return ControllerConfig{}, fmt.Errorf("failed to read time (0x%X): %w", regRealTimeClock, err)
	}
	if len(data) < 6 {
		return ControllerConfig{}, fmt.Errorf("insufficient time data: expected 6 bytes, got %d", len(data))
	}

	year := int(data[4]) + 2000
	timeStr := fmt.Sprintf("%d-%d-%d %02d:%02d:%02d",
		data[2], data[5], year, data[3], data[0], data[1])

	// Read voltage parameters
	data, err = sc.modbusClient.ReadHoldingRegisters(ctx, regOverVoltDisconnect, 12)
	if err != nil {
		return ControllerConfig{}, fmt.Errorf("failed to read voltage parameters (0x%X): %w", regOverVoltDisconnect, err)
	}
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
		return ControllerConfig{}, fmt.Errorf("failed to read equalization cycle (0x%X): %w", regEqualizationChargingCycle, err)
	}
	if len(data) < 2 {
		return ControllerConfig{}, fmt.Errorf("insufficient equalization cycle data: expected 2 bytes, got %d", len(data))
	}
	equalizationCycle := binary.BigEndian.Uint16(data[0:2])

	// Read durations
	data, err = sc.modbusClient.ReadHoldingRegisters(ctx, regEqualizationChargingTime, 2)
	if err != nil {
		return ControllerConfig{}, fmt.Errorf("failed to read durations (0x%X): %w", regEqualizationChargingTime, err)
	}
	if len(data) < 4 {
		return ControllerConfig{}, fmt.Errorf("insufficient duration data: expected 4 bytes, got %d", len(data))
	}
	equalizationDuration := binary.BigEndian.Uint16(data[0:2])
	boostDuration := binary.BigEndian.Uint16(data[2:4])

	// Read temperature limits
	data, err = sc.modbusClient.ReadHoldingRegisters(ctx, regBatteryTempUpperLimit, 4)
	if err != nil {
		return ControllerConfig{}, fmt.Errorf("failed to read temperature limits (0x%X): %w", regBatteryTempUpperLimit, err)
	}
	if len(data) < 8 {
		return ControllerConfig{}, fmt.Errorf("insufficient temperature data: expected 8 bytes, got %d", len(data))
	}
	batteryTempUpperLimit := float32(int16(binary.BigEndian.Uint16(data[0:2]))) / voltageDivisor
	batteryTempLowerLimit := float32(int16(binary.BigEndian.Uint16(data[2:4]))) / voltageDivisor
	controllerTempUpperLimit := float32(int16(binary.BigEndian.Uint16(data[4:6]))) / voltageDivisor
	controllerTempLowerLimit := float32(int16(binary.BigEndian.Uint16(data[6:8]))) / voltageDivisor

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

func batteryTypeToInt(batteryType string) uint16 {
	switch batteryType {
	case "sealed":
		return 1
	case "gel":
		return 2
	case "flooded":
		return 3
	case "userDefined":
		return 4
	default:
		return 0
	}
}

func batteryTypeToString(batteryType uint16) string {
	switch batteryType {
	case 1:
		return "sealed"
	case 2:
		return "gel"
	case 3:
		return "flooded"
	case 4:
		return "userDefined"
	default:
		return "unknown"
	}
}

func (sc *Configurer) writeSingle(c *gin.Context, address, value uint16, description string) error {
	log.Info(fmt.Sprintf("Setting %v of %v to controller", description, value))
	_, err := sc.modbusClient.WriteSingleRegister(c.Request.Context(), address, value)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to write %v of %v to controller", description, value)
		log.Warn(errorMessage, err.Error())
		if sc.prometheusCollector != nil {
			sc.prometheusCollector.IncrementWriteFailures()
		}
		return fmt.Errorf("%s: %w", errorMessage, err)
	}
	return nil
}

func (sc *Configurer) ConfigPatch() gin.HandlerFunc {
	return func(c *gin.Context) {
		var config ControllerConfig
		err := c.BindJSON(&config)
		if err != nil {
			log.Warn("Config patch bad json request", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userDefined := false
		if config.BatteryType != "" {
			batteryType := batteryTypeToInt(config.BatteryType)
			if err := sc.writeSingle(c, regBatteryType, batteryType, "battery type"); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			// Invalidate cache when battery type is changed
			sc.invalidateCache()

			userDefined = batteryType == batteryTypeUserDefined
		} else {
			data, err := sc.modbusClient.ReadHoldingRegisters(c.Request.Context(), regBatteryType, 1)
			if err != nil {
				log.Warn("Failed to read battery type", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read battery type"})
				return
			}
			userDefined = binary.BigEndian.Uint16(data[0:2]) == batteryTypeUserDefined
		}

		if userDefined {
			// Get current configuration for validation
			currentConfig, err := sc.getCachedConfig(c.Request.Context())
			if err != nil {
				log.Warn("Failed to read current config for validation", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read current configuration"})
				return
			}

			// Create a copy of the current config to apply proposed changes
			proposedConfig := *currentConfig

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

			// Validation passed, proceed with writes
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

			if config.ChargingLimitVoltage > 0 {
				chargingLimitVoltage := uint16(config.ChargingLimitVoltage * voltageDivisor)
				if err := sc.writeSingle(c, regChargingLimitVoltage, chargingLimitVoltage, "charging limit voltage"); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
			}

			if config.EqualizationVoltage > 0 {
				equalizationVoltage := uint16(config.EqualizationVoltage * voltageDivisor)
				if err := sc.writeSingle(c, regEqualizationVoltage, equalizationVoltage, "equalization voltage"); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
			}

			if config.BoostVoltage > 0 {
				boostVoltage := uint16(config.BoostVoltage * voltageDivisor)
				if err := sc.writeSingle(c, regBoostVoltage, boostVoltage, "boost voltage"); err != nil {
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

			if config.FloatVoltage > 0 {
				floatVoltage := uint16(config.FloatVoltage * voltageDivisor)
				if err := sc.writeSingle(c, regFloatVoltage, floatVoltage, "float voltage"); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
			}

			if config.BoostReconnectChargingVoltage > 0 {
				boostReconnectChargingVoltage := uint16(config.BoostReconnectChargingVoltage * voltageDivisor)
				if err := sc.writeSingle(c, regBoostReconnectVoltage, boostReconnectChargingVoltage, "boost reconnect charging voltage"); err != nil {
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

// BatteryProfileGet returns the battery profile (type and capacity)
func (sc *Configurer) BatteryProfileGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		config, err := sc.getCachedConfig(c.Request.Context())
		if err != nil {
			log.Warn("Failed to get battery profile", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		profile := gin.H{
			"batteryType":     config.BatteryType,
			"batteryCapacity": config.BatteryCapacity,
		}
		c.JSON(http.StatusOK, profile)
	}
}

// BatteryProfilePatch updates the battery profile (only fields present in request)
func (sc *Configurer) BatteryProfilePatch() gin.HandlerFunc {
	return func(c *gin.Context) {
		var rawData map[string]json.RawMessage
		if err := c.BindJSON(&rawData); err != nil {
			log.Warn("Battery profile patch bad json request", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Track if any writes succeeded
		writeSucceeded := false

		// Check and write battery type if present
		if batteryTypeRaw, ok := rawData["batteryType"]; ok {
			var batteryType string
			if err := json.Unmarshal(batteryTypeRaw, &batteryType); err == nil {
				batteryTypeInt := batteryTypeToInt(batteryType)
				if err := sc.writeSingle(c, regBatteryType, batteryTypeInt, "battery type"); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				writeSucceeded = true
			}
		}

		// Check and write battery capacity if present
		if batteryCapacityRaw, ok := rawData["batteryCapacity"]; ok {
			var batteryCapacity uint16
			if err := json.Unmarshal(batteryCapacityRaw, &batteryCapacity); err == nil {
				if err := sc.writeSingle(c, regBatteryCapacity, batteryCapacity, "battery capacity"); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				writeSucceeded = true
			}
		}

		// Invalidate cache after successful write
		if writeSucceeded {
			sc.invalidateCache()
		}

		// Return updated profile (this will fetch fresh data from device)
		config, err := sc.getCachedConfig(c.Request.Context())
		if err != nil {
			log.Warn("Failed to retrieve updated profile after write", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Profile updated but failed to read back"})
			return
		}
		profile := gin.H{
			"batteryType":     config.BatteryType,
			"batteryCapacity": config.BatteryCapacity,
		}
		c.JSON(http.StatusOK, profile)
	}
}

// validateVoltageParameters validates voltage relationships according to modbus register documentation
// Returns an error if any of the 4 voltage relationship rules are violated
func validateVoltageParameters(config *ControllerConfig) error {
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

// ChargingParametersGet returns all charging parameters
func (sc *Configurer) ChargingParametersGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		config, err := sc.getCachedConfig(c.Request.Context())
		if err != nil {
			log.Warn("Failed to get charging parameters", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		params := gin.H{
			"boostDuration":                    config.BoostDuration,
			"equalizationCycle":                config.EqualizationCycle,
			"equalizationDuration":             config.EqualizationDuration,
			"boostVoltage":                     config.BoostVoltage,
			"boostReconnectChargingVoltage":    config.BoostReconnectChargingVoltage,
			"floatVoltage":                     config.FloatVoltage,
			"equalizationVoltage":              config.EqualizationVoltage,
			"chargingLimitVoltage":             config.ChargingLimitVoltage,
			"overVoltDisconnectVoltage":        config.OverVoltDisconnectVoltage,
			"overVoltReconnectVoltage":         config.OverVoltReconnectVoltage,
			"lowVoltDisconnectVoltage":         config.LowVoltDisconnectVoltage,
			"lowVoltReconnectVoltage":          config.LowVoltReconnectVoltage,
			"underVoltWarningVoltage":          config.UnderVoltWarningVoltage,
			"underVoltWarningReconnectVoltage": config.UnderVoltReconnectVoltage,
			"dischargingLimitVoltage":          config.DischargingLimitVoltage,
			"batteryTempUpperLimit":            config.BatteryTempUpperLimit,
			"batteryTempLowerLimit":            config.BatteryTempLowerLimit,
			"controllerTempUpperLimit":         config.ControllerTempUpperLimit,
			"controllerTempLowerLimit":         config.ControllerTempLowerLimit,
		}
		c.JSON(http.StatusOK, params)
	}
}

// ChargingParametersPatch updates charging parameters (only fields present in request, only if userDefined)
func (sc *Configurer) ChargingParametersPatch() gin.HandlerFunc {
	return func(c *gin.Context) {
		// First check if battery type is userDefined
		data, err := sc.modbusClient.ReadHoldingRegisters(c.Request.Context(), regBatteryType, 1)
		if err != nil {
			log.Warn("Failed to read battery type", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read battery type"})
			return
		}

		batteryType := binary.BigEndian.Uint16(data[0:2])
		if batteryType != batteryTypeUserDefined {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Charging parameters can only be modified when battery type is 'userDefined'"})
			return
		}

		var rawData map[string]json.RawMessage
		if err := c.BindJSON(&rawData); err != nil {
			log.Warn("Charging parameters patch bad json request", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get current configuration from device
		currentConfig, err := sc.getCachedConfig(c.Request.Context())
		if err != nil {
			log.Warn("Failed to read current config for validation", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read current configuration"})
			return
		}

		// Create a copy of the current config to apply proposed changes
		proposedConfig := *currentConfig

		// Apply requested changes to proposed config for validation
		if val, ok := rawData["chargingLimitVoltage"]; ok {
			var chargingLimitVoltage float32
			if err := json.Unmarshal(val, &chargingLimitVoltage); err == nil {
				proposedConfig.ChargingLimitVoltage = chargingLimitVoltage
			}
		}

		if val, ok := rawData["equalizationVoltage"]; ok {
			var equalizationVoltage float32
			if err := json.Unmarshal(val, &equalizationVoltage); err == nil {
				proposedConfig.EqualizationVoltage = equalizationVoltage
			}
		}

		if val, ok := rawData["boostVoltage"]; ok {
			var boostVoltage float32
			if err := json.Unmarshal(val, &boostVoltage); err == nil {
				proposedConfig.BoostVoltage = boostVoltage
			}
		}

		if val, ok := rawData["floatVoltage"]; ok {
			var floatVoltage float32
			if err := json.Unmarshal(val, &floatVoltage); err == nil {
				proposedConfig.FloatVoltage = floatVoltage
			}
		}

		if val, ok := rawData["boostReconnectChargingVoltage"]; ok {
			var boostReconnectChargingVoltage float32
			if err := json.Unmarshal(val, &boostReconnectChargingVoltage); err == nil {
				proposedConfig.BoostReconnectChargingVoltage = boostReconnectChargingVoltage
			}
		}

		// Validate the proposed configuration
		if err := validateVoltageParameters(&proposedConfig); err != nil {
			log.Warn("Voltage parameter validation failed", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Track if any writes succeeded
		writeSucceeded := false

		// Write each field that is present in the request
		if val, ok := rawData["equalizationCycle"]; ok {
			var equalizationCycle uint16
			if err := json.Unmarshal(val, &equalizationCycle); err == nil {
				if err := sc.writeSingle(c, regEqualizationChargingCycle, equalizationCycle, "equalization cycle"); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				writeSucceeded = true
			}
		}

		if val, ok := rawData["equalizationDuration"]; ok {
			var equalizationDuration uint16
			if err := json.Unmarshal(val, &equalizationDuration); err == nil {
				if err := sc.writeSingle(c, regEqualizationChargingTime, equalizationDuration, "equalization duration"); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				writeSucceeded = true
			}
		}

		if val, ok := rawData["chargingLimitVoltage"]; ok {
			var chargingLimitVoltage float32
			if err := json.Unmarshal(val, &chargingLimitVoltage); err == nil {
				if err := sc.writeSingle(c, regChargingLimitVoltage, uint16(chargingLimitVoltage*voltageDivisor), "charging limit voltage"); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				writeSucceeded = true
			}
		}

		if val, ok := rawData["equalizationVoltage"]; ok {
			var equalizationVoltage float32
			if err := json.Unmarshal(val, &equalizationVoltage); err == nil {
				if err := sc.writeSingle(c, regEqualizationVoltage, uint16(equalizationVoltage*voltageDivisor), "equalization voltage"); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				writeSucceeded = true
			}
		}

		if val, ok := rawData["boostVoltage"]; ok {
			var boostVoltage float32
			if err := json.Unmarshal(val, &boostVoltage); err == nil {
				if err := sc.writeSingle(c, regBoostVoltage, uint16(boostVoltage*voltageDivisor), "boost voltage"); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				writeSucceeded = true
			}
		}

		if val, ok := rawData["boostDuration"]; ok {
			var boostDuration uint16
			if err := json.Unmarshal(val, &boostDuration); err == nil {
				if err := sc.writeSingle(c, regBoostChargingTime, boostDuration, "boost duration"); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				writeSucceeded = true
			}
		}

		if val, ok := rawData["floatVoltage"]; ok {
			var floatVoltage float32
			if err := json.Unmarshal(val, &floatVoltage); err == nil {
				if err := sc.writeSingle(c, regFloatVoltage, uint16(floatVoltage*voltageDivisor), "float voltage"); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				writeSucceeded = true
			}
		}

		if val, ok := rawData["boostReconnectChargingVoltage"]; ok {
			var boostReconnectChargingVoltage float32
			if err := json.Unmarshal(val, &boostReconnectChargingVoltage); err == nil {
				if err := sc.writeSingle(c, regBoostReconnectVoltage, uint16(boostReconnectChargingVoltage*voltageDivisor), "boost reconnect charging voltage"); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				writeSucceeded = true
			}
		}

		// Invalidate cache after successful write
		if writeSucceeded {
			sc.invalidateCache()
		}

		// Return updated parameters (this will fetch fresh data from device)
		config, err := sc.getCachedConfig(c.Request.Context())
		if err != nil {
			log.Warn("Failed to retrieve updated charging parameters after write", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Parameters updated but failed to read back"})
			return
		}
		params := gin.H{
			"boostDuration":                    config.BoostDuration,
			"equalizationCycle":                config.EqualizationCycle,
			"equalizationDuration":             config.EqualizationDuration,
			"boostVoltage":                     config.BoostVoltage,
			"boostReconnectChargingVoltage":    config.BoostReconnectChargingVoltage,
			"floatVoltage":                     config.FloatVoltage,
			"equalizationVoltage":              config.EqualizationVoltage,
			"chargingLimitVoltage":             config.ChargingLimitVoltage,
			"overVoltDisconnectVoltage":        config.OverVoltDisconnectVoltage,
			"overVoltReconnectVoltage":         config.OverVoltReconnectVoltage,
			"lowVoltDisconnectVoltage":         config.LowVoltDisconnectVoltage,
			"lowVoltReconnectVoltage":          config.LowVoltReconnectVoltage,
			"underVoltWarningVoltage":          config.UnderVoltWarningVoltage,
			"underVoltWarningReconnectVoltage": config.UnderVoltReconnectVoltage,
			"dischargingLimitVoltage":          config.DischargingLimitVoltage,
			"batteryTempUpperLimit":            config.BatteryTempUpperLimit,
			"batteryTempLowerLimit":            config.BatteryTempLowerLimit,
			"controllerTempUpperLimit":         config.ControllerTempUpperLimit,
			"controllerTempLowerLimit":         config.ControllerTempLowerLimit,
		}
		c.JSON(http.StatusOK, params)
	}
}

// TimeGet returns the current time from the controller
func (sc *Configurer) TimeGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := sc.modbusClient.ReadHoldingRegisters(c.Request.Context(), regRealTimeClock, 3)
		if err != nil {
			log.Warn("Failed to read time from controller", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read time from controller"})
			return
		}
		if len(data) < 6 {
			log.Warnf("Insufficient time data: expected 6 bytes, got %d", len(data))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Insufficient time data from controller"})
			return
		}

		// Parse time: min, sec, day, hour, month, year
		year := int(data[4]) + 2000
		month := time.Month(data[5])
		day := int(data[2])
		hour := int(data[3])
		minute := int(data[0])
		second := int(data[1])

		controllerTime := time.Date(year, month, day, hour, minute, second, 0, time.UTC)

		c.JSON(http.StatusOK, gin.H{"time": controllerTime})
	}
}

// TimePatch updates the controller time
func (sc *Configurer) TimePatch() gin.HandlerFunc {
	return func(c *gin.Context) {
		var timeConfig TimeConfig
		if err := c.BindJSON(&timeConfig); err != nil {
			log.Warn("Time patch bad json request", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// min, sec, day, hour, year, month
		data := []byte{
			byte(timeConfig.Time.Minute()),
			byte(timeConfig.Time.Second()),
			byte(timeConfig.Time.Day()),
			byte(timeConfig.Time.Hour()),
			byte(timeConfig.Time.Year() - 2000),
			byte(timeConfig.Time.Month()),
		}

		_, err := sc.modbusClient.WriteMultipleRegisters(c.Request.Context(), regRealTimeClock, 3, data)
		if err != nil {
			log.Warn("Failed to write time to controller", err)
			if sc.prometheusCollector != nil {
				sc.prometheusCollector.IncrementWriteFailures()
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Invalidate cache after time write
		sc.invalidateCache()

		// Return the updated time
		data, err = sc.modbusClient.ReadHoldingRegisters(c.Request.Context(), regRealTimeClock, 3)
		if err != nil {
			log.Warn("Failed to read time from controller after write", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read time from controller"})
			return
		}
		if len(data) < 6 {
			log.Warnf("Insufficient time data after write: expected 6 bytes, got %d", len(data))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Insufficient time data from controller"})
			return
		}

		year := int(data[4]) + 2000
		month := time.Month(data[5])
		day := int(data[2])
		hour := int(data[3])
		minute := int(data[0])
		second := int(data[1])

		controllerTime := time.Date(year, month, day, hour, minute, second, 0, time.UTC)

		c.JSON(http.StatusOK, gin.H{"time": controllerTime})
	}
}
