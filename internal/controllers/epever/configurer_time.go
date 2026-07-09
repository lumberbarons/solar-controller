package epever

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type TimeConfig struct {
	Time time.Time `json:"time"`
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
		// Allow device time to fully commit changes to EEPROM before reading back
		time.Sleep(500 * time.Millisecond)

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
