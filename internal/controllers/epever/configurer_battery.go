package epever

import (
	"encoding/json"
	"fmt"
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

func parseBatteryType(batteryType string) (uint16, error) {
	switch batteryType {
	case "userDefined":
		return 0, nil
	case "sealed":
		return 1, nil
	case "gel":
		return 2, nil
	case "flooded":
		return 3, nil
	default:
		return 0, fmt.Errorf("unknown battery type %q (expected userDefined, sealed, gel, or flooded)", batteryType)
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
		if err := bindJSONBounded(c, &rawData); err != nil {
			log.Warn("Battery profile patch bad json request", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Parse and validate every provided field before writing anything,
		// so a malformed or out-of-range request performs no writes
		var batteryType *uint16
		if raw, ok := rawData["batteryType"]; ok {
			var typeName string
			if err := json.Unmarshal(raw, &typeName); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid batteryType: %v", err)})
				return
			}
			typeValue, err := parseBatteryType(typeName)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			batteryType = &typeValue
		}

		var batteryCapacity *uint16
		if raw, ok := rawData["batteryCapacity"]; ok {
			var capacity uint16
			if err := json.Unmarshal(raw, &capacity); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid batteryCapacity: %v", err)})
				return
			}
			if capacity < minBatteryCapacityAh || capacity > maxBatteryCapacityAh {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("batteryCapacity (%d) out of range [%d, %d] Ah", capacity, minBatteryCapacityAh, maxBatteryCapacityAh)})
				return
			}
			batteryCapacity = &capacity
		}

		var tempCompCoefficient *uint16
		if raw, ok := rawData["tempCompCoefficient"]; ok {
			var coefficient float32
			if err := json.Unmarshal(raw, &coefficient); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid tempCompCoefficient: %v", err)})
				return
			}
			if coefficient < minTempCompCoefficient || coefficient > maxTempCompCoefficient {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("tempCompCoefficient (%.2f) out of range [%.1f, %.1f]", coefficient, float64(minTempCompCoefficient), float64(maxTempCompCoefficient))})
				return
			}
			registerValue := uint16(coefficient * voltageDivisor)
			tempCompCoefficient = &registerValue
		}

		// Track if any writes succeeded
		writeSucceeded := false

		if batteryType != nil {
			if err := sc.writeSingle(c, regBatteryType, *batteryType, "battery type"); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			writeSucceeded = true
		}

		if batteryCapacity != nil {
			if err := sc.writeSingle(c, regBatteryCapacity, *batteryCapacity, "battery capacity"); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			writeSucceeded = true
		}

		if tempCompCoefficient != nil {
			if err := sc.writeSingle(c, regTempCompCoefficient, *tempCompCoefficient, "temperature compensation coefficient"); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			writeSucceeded = true
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
