package config

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/goburrow/modbus"
	"net/http"
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

func (sc *SolarConfigurer) TimeHandlerGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		data, _ := sc.modbusClient.ReadHoldingRegisters(0x9013, 3)

		// min, sec, day, hour, year, month
		date := fmt.Sprintf("%02d:%02d:%02d %d-%d-%d",
			data[3], data[0], data[1], data[2], data[5], data[4])

		c.String(http.StatusOK, date)
	}
}

