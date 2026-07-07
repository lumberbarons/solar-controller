package epever

import (
	"encoding/binary"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

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
		voltageParamsPresent := false

		// Check if any voltage parameters are present in the request
		voltageFields := []string{"chargingLimitVoltage", "equalizationVoltage", "boostVoltage",
			"floatVoltage", "boostReconnectChargingVoltage"}
		for _, field := range voltageFields {
			if _, ok := rawData[field]; ok {
				voltageParamsPresent = true
				break
			}
		}

		// If any voltage parameters are present, write the entire voltage block
		if voltageParamsPresent {
			if err := sc.writeVoltageParametersBlock(c, &proposedConfig); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			writeSucceeded = true
		}

		// Write non-voltage parameters individually
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
