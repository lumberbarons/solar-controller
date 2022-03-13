package config

import (
	"encoding/binary"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/goburrow/modbus"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"strings"
)

type SolarConfigurer struct {
	modbusClient modbus.Client
}

func NewSolarConfigurer(client modbus.Client) *SolarConfigurer {
	return &SolarConfigurer{
		modbusClient: client,
	}
}

type ControllerConfig struct {
	Time            string `json:"time"`
	BatteryType     string `json:"batteryType"`
	BatteryCapacity uint16 `json:"batteryCapacity"`

	FloatVoltage		  float32 `json:"floatVoltage"`
	EqualizationVoltage   float32 `json:"equalizationVoltage"`
	EqualizationCycle     uint16  `json:"equalizationCycle"`
	EqualizationDuration  uint16  `json:"equalizationDuration"`
	BoostVoltage		  float32 `json:"boostVoltage"`
	BoostReconnectVoltage float32 `json:"boostReconnectVoltage"`
	BoostDuration         uint16  `json:"boostDuration"`
}

type ControllerQuery struct {
	Register int    `json:"register"`
	Address  string `json:"address"`
	Result	 uint16 `json:"result"`
}

func (sc *SolarConfigurer) ConfigGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		data, _ := sc.modbusClient.ReadHoldingRegisters(0x9000, 2)

		batteryType := binary.BigEndian.Uint16(data[0:2])
		batteryCapacity := binary.BigEndian.Uint16(data[2:4])

		data, _ = sc.modbusClient.ReadHoldingRegisters(0x9013, 3)

		year := int(data[4]) + 2000
		time := fmt.Sprintf("%d-%d-%d %02d:%02d:%02d",
			data[2], data[5], year, data[3], data[0], data[1])

		data, _ = sc.modbusClient.ReadHoldingRegisters(0x9006, 4)
		equalizationVoltage := float32(binary.BigEndian.Uint16(data[0:2])) / 100
		boostVoltage := float32(binary.BigEndian.Uint16(data[2:4])) / 100
		floatVoltage := float32(binary.BigEndian.Uint16(data[4:6])) / 100
		boostReconnectVoltage := float32(binary.BigEndian.Uint16(data[6:8])) / 100

		data, _ = sc.modbusClient.ReadHoldingRegisters(0x9016, 1)
		equalizationCycle := binary.BigEndian.Uint16(data[0:2])

		data, _ = sc.modbusClient.ReadHoldingRegisters(0x906B, 2)
		equalizationDuration := binary.BigEndian.Uint16(data[0:2])
		boostDuration := binary.BigEndian.Uint16(data[2:4])

		c.JSON(http.StatusOK, ControllerConfig{Time: time,
			BatteryType: batteryTypeToString(batteryType),
			BatteryCapacity: batteryCapacity, 
			EqualizationVoltage: equalizationVoltage,
			EqualizationCycle: equalizationCycle,
			BoostVoltage: boostVoltage, FloatVoltage: floatVoltage,
			BoostReconnectVoltage: boostReconnectVoltage,
			BoostDuration: boostDuration,
			EqualizationDuration: equalizationDuration})
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

func (sc *SolarConfigurer) ConfigPatch() gin.HandlerFunc {
	return func(c *gin.Context) {
		/* currentTime := time.Now()

		// min, sec, day, hour, year, month
		data := []byte {byte(currentTime.Minute()), byte(currentTime.Second()),
			byte(currentTime.Day()), byte(currentTime.Hour()),
			byte(currentTime.Year() - 2000), byte(currentTime.Month())}

		result, _ := sc.modbusClient.WriteMultipleRegisters(0x9013, 3, data)
		log.Debugf("time write result: %x", result) */

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

		log.Info("Query result: ", result)

		query.Result = binary.BigEndian.Uint16(result)
		c.JSON(http.StatusOK, query)
	}
}

