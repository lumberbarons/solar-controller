package config

import (
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/goburrow/modbus"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type SolarConfigurer struct {
	modbusClient modbus.Client
}

func NewSolarConfigurer(client modbus.Client) *SolarConfigurer {
	return &SolarConfigurer{
		modbusClient: client,
	}
}

type ControllerTime struct {
	Time string `json:"time"`
}

type ControllerQuery struct {
	Register int     `json:"register"`
	Address  string  `json:"address"`
	Result	 string `json:"result"`
}

func (sc *SolarConfigurer) TimeGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		data, _ := sc.modbusClient.ReadHoldingRegisters(0x9013, 3)

		// min, sec, day, hour, year, month
		date := fmt.Sprintf("%02d:%02d:%02d %d-%d-%d",
			data[3], data[0], data[1], data[2], data[5], data[4])

		c.JSON(http.StatusOK, ControllerTime{Time: date})
	}
}

func (sc *SolarConfigurer) TimePut() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentTime := time.Now()

		// min, sec, day, hour, year, month
		data := []byte {byte(currentTime.Minute()), byte(currentTime.Second()),
			byte(currentTime.Day()), byte(currentTime.Hour()),
			byte(currentTime.Year() - 2000), byte(currentTime.Month())}

		result, _ := sc.modbusClient.WriteMultipleRegisters(0x9013, 3, data)
		log.Debugf("time write result: %x", result)

		c.Status(http.StatusOK)
	}
}

func (sc *SolarConfigurer) QueryPost() gin.HandlerFunc {
	return func(c *gin.Context) {
		var query ControllerQuery
		err := c.BindJSON(&query)
		if err != nil {
			log.Warn("Query bad json request")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cleaned := strings.Replace(query.Address, "0x", "", -1)
		address, err := strconv.ParseUint(cleaned, 16, 16)
		if err != nil {
			log.Warn("Query bad address: ", query.Address, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var result []byte
		if query.Register == 1 {
			result,_ = sc.modbusClient.ReadCoils(uint16(address), 1)
		} else if query.Register == 2 {
			result,_ = sc.modbusClient.ReadDiscreteInputs(uint16(address), 1)
		} else if query.Register == 3 {
			result,_ = sc.modbusClient.ReadHoldingRegisters(uint16(address), 1)
		} else if query.Register == 4 {
			result,_ = sc.modbusClient.ReadInputRegisters(uint16(address), 1)
		} else {
			log.Warn("Query bad register: ", query.Register)
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown register"})
			return
		}

		log.Info("query result: ", result)
		query.Result = "0x" + hex.EncodeToString(result)

		c.JSON(http.StatusOK, query)
	}
}

