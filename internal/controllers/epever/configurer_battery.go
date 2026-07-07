package epever

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// BatteryProfile contains battery identity settings
type BatteryProfile struct {
	BatteryType         *string  `json:"batteryType,omitempty"`
	BatteryCapacity     *uint16  `json:"batteryCapacity,omitempty"`
	TempCompCoefficient *float32 `json:"tempCompCoefficient,omitempty"`
}

func batteryTypeToInt(batteryType string) uint16 {
	switch batteryType {
	case "userDefined":
		return 0
	case "sealed":
		return 1
	case "gel":
		return 2
	case "flooded":
		return 3
	default:
		return 4
	}
}

func batteryTypeToString(batteryType uint16) string {
	switch batteryType {
	case 0:
		return "userDefined"
	case 1:
		return "sealed"
	case 2:
		return "gel"
	case 3:
		return "flooded"
	default:
		return "unknown"
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
			"batteryType":         config.BatteryType,
			"batteryCapacity":     config.BatteryCapacity,
			"tempCompCoefficient": config.TempCompCoefficient,
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

		// Check and write temperature compensation coefficient if present
		if tempCompCoefficientRaw, ok := rawData["tempCompCoefficient"]; ok {
			var tempCompCoefficient float32
			if err := json.Unmarshal(tempCompCoefficientRaw, &tempCompCoefficient); err == nil {
				tempCompCoefficientInt := uint16(tempCompCoefficient * voltageDivisor)
				if err := sc.writeSingle(c, regTempCompCoefficient, tempCompCoefficientInt, "temperature compensation coefficient"); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				writeSucceeded = true
			}
		}

		// Invalidate cache after successful write
		if writeSucceeded {
			sc.invalidateCache()
			// Allow device time to fully commit all changes to EEPROM before reading back
			time.Sleep(500 * time.Millisecond)
		}

		// Return updated profile (this will fetch fresh data from device)
		config, err := sc.getCachedConfig(c.Request.Context())
		if err != nil {
			log.Warn("Failed to retrieve updated profile after write", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Profile updated but failed to read back"})
			return
		}
		profile := gin.H{
			"batteryType":         config.BatteryType,
			"batteryCapacity":     config.BatteryCapacity,
			"tempCompCoefficient": config.TempCompCoefficient,
		}
		c.JSON(http.StatusOK, profile)
	}
}
