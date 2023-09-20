package epever

import (
	"encoding/binary"
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Configurer struct {
	modbusClient *ModbusClient
}

func NewConfigurer(client *ModbusClient) *Configurer {
	return &Configurer{
		modbusClient: client,
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

type ControllerQuery struct {
	Register int    `json:"register"`
	Address  string `json:"address"`
	Result   uint16 `json:"result"`
}

func (sc *Configurer) ConfigGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		config, _ := sc.getConfig()
		c.JSON(http.StatusOK, config)
	}
}

func (sc *Configurer) getFloatValue(data []byte, index int) float32 {
	offset := index * 2
	return float32(binary.BigEndian.Uint16(data[offset:offset+2])) / 100
}

func (sc *Configurer) getConfig() (ControllerConfig, error) {
	data, _ := sc.modbusClient.ReadHoldingRegisters(0x9000, 3)

	batteryType := binary.BigEndian.Uint16(data[0:2])
	batteryCapacity := binary.BigEndian.Uint16(data[2:4])
	tempCompCoefficient := sc.getFloatValue(data, 2)

	data, _ = sc.modbusClient.ReadHoldingRegisters(0x9013, 3)

	year := int(data[4]) + 2000
	time := fmt.Sprintf("%d-%d-%d %02d:%02d:%02d",
		data[2], data[5], year, data[3], data[0], data[1])

	data, _ = sc.modbusClient.ReadHoldingRegisters(0x9003, 12)
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

	data, _ = sc.modbusClient.ReadHoldingRegisters(0x9016, 1)
	equalizationCycle := binary.BigEndian.Uint16(data[0:2])

	data, _ = sc.modbusClient.ReadHoldingRegisters(0x906B, 2)
	equalizationDuration := binary.BigEndian.Uint16(data[0:2])
	boostDuration := binary.BigEndian.Uint16(data[2:4])

	data, _ = sc.modbusClient.ReadHoldingRegisters(0x9017, 4)
	batteryTempUpperLimit := float32(int16(binary.BigEndian.Uint16(data[0:2]))) / 100
	batteryTempLowerLimit := float32(int16(binary.BigEndian.Uint16(data[2:4]))) / 100
	controllerTempUpperLimit := float32(int16(binary.BigEndian.Uint16(data[4:6]))) / 100
	controllerTempLowerLimit := float32(int16(binary.BigEndian.Uint16(data[6:8]))) / 100

	return ControllerConfig{
		Time:                          time,
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

func (sc *Configurer) writeSingle(c *gin.Context, address uint16, value uint16, description string) {
	log.Info(fmt.Sprintf("Setting %v of %v to controller", description, value))
	_, err := sc.modbusClient.WriteSingleRegister(address, value)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to write %v of %v to controller", description, value)
		log.Warn(errorMessage, err.Error())

		c.JSON(http.StatusBadRequest, gin.H{"message": errorMessage})
		return
	}
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

		currentTime := time.Now()

		// min, sec, day, hour, year, month
		data := []byte{byte(currentTime.Minute()), byte(currentTime.Second()),
			byte(currentTime.Day()), byte(currentTime.Hour()),
			byte(currentTime.Year() - 2000), byte(currentTime.Month())}

		_, err = sc.modbusClient.WriteMultipleRegisters(0x9013, 3, data)
		if err != nil {
			log.Warn("Failed to write date to controller")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userDefined := false
		if config.BatteryType != "" {
			batteryType := batteryTypeToInt(config.BatteryType)
			sc.writeSingle(c, 0x9000, batteryType, "battery type")

			userDefined = batteryType == 4
		} else {
			data, _ := sc.modbusClient.ReadHoldingRegisters(0x9000, 1)
			userDefined = binary.BigEndian.Uint16(data[0:2]) == 4
		}

		if userDefined {
			if config.EqualizationCycle > 0 {
				sc.writeSingle(c, 0x9016, config.EqualizationCycle, "equalization cycle")
			}

			if config.EqualizationDuration > 0 {
				sc.writeSingle(c, 0x906B, config.EqualizationDuration, "equalization duration")
			}

			if config.ChargingLimitVoltage > 0 {
				chargingLimitVoltage := uint16(config.ChargingLimitVoltage * 100)
				sc.writeSingle(c, 0x9004, chargingLimitVoltage, "charging limit voltage")
			}

			if config.EqualizationVoltage > 0 {
				equalizationVoltage := uint16(config.EqualizationVoltage * 100)
				sc.writeSingle(c, 0x9006, equalizationVoltage, "equalization voltage")
			}

			if config.BoostVoltage > 0 {
				boostVoltage := uint16(config.BoostVoltage * 100)
				sc.writeSingle(c, 0x9007, boostVoltage, "boost voltage")
			}

			if config.BoostDuration > 0 {
				sc.writeSingle(c, 0x906C, config.BoostDuration, "boost duration")
			}

			if config.FloatVoltage > 0 {
				floatVoltage := uint16(config.FloatVoltage * 100)
				sc.writeSingle(c, 0x9008, floatVoltage, "float voltage")
			}
		}

		newConfig, _ := sc.getConfig()
		c.JSON(http.StatusOK, newConfig)
	}
}

func (sc *Configurer) QueryPost() gin.HandlerFunc {
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
			result, _ = sc.modbusClient.ReadCoils(uint16(address), 1)
		} else if query.Register == 2 {
			result, _ = sc.modbusClient.ReadDiscreteInputs(uint16(address), 1)
		} else if query.Register == 3 {
			result, _ = sc.modbusClient.ReadHoldingRegisters(uint16(address), 1)
		} else if query.Register == 4 {
			result, _ = sc.modbusClient.ReadInputRegisters(uint16(address), 1)
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
