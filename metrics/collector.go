package metrics

import (
	"encoding/binary"
	"github.com/goburrow/modbus"
	log "github.com/sirupsen/logrus"
)

type ControllerStatus struct {
	ArrayVoltage           float32   `json:"arrayVoltage"`
	ArrayCurrent           float32   `json:"arrayCurrent"`
	ArrayPower             float32   `json:"arrayPower"`
	BatteryVoltage         float32   `json:"batteryVoltage"`
	BatterySOC             int32     `json:"batterySoc"`
	BatteryTemp            float32   `json:"batteryTemp"`
	BatteryMaxVoltage      float32   `json:"batteryMaxVoltage"`
	BatteryMinVoltage      float32   `json:"batteryMinVoltage"`
	DeviceTemp             float32   `json:"deviceTemp"`
	EnergyGeneratedDaily   float32   `json:"energyGeneratedDaily"`
	EnergyGeneratedMonthly float32   `json:"energyGeneratedMonthly"`
	EnergyGeneratedAnnual  float32   `json:"energyGeneratedAnnually"`
	EnergyGeneratedTotal   float32   `json:"energyGeneratedTotal"`
}

func getStatus(client modbus.Client) (c ControllerStatus, err error) {
	c.ArrayVoltage = getValue(client, 0x3100) / 100
	c.ArrayCurrent = getValue(client, 0x3101) / 100
	c.BatteryVoltage = getValue(client, 0x3104) / 100
	c.BatterySOC = int32(getValue(client, 0x311A))

	c.BatteryMaxVoltage = getValue(client, 0x3302) / 100
	c.BatteryMinVoltage = getValue(client, 0x3303) / 100

	c.ArrayPower = getValue32(client, 0x3102) / 100

	c.EnergyGeneratedDaily = getValue32(client, 0x330C) / 100
	c.EnergyGeneratedMonthly = getValue32(client, 0x330E) / 100
	c.EnergyGeneratedAnnual = getValue32(client, 0x3310) / 100
	c.EnergyGeneratedTotal = getValue32(client, 0x3312) / 100

	bt := getValue(client, 0x3110)
	if bt > 32768 {
		bt = bt - 65536
	}
	c.BatteryTemp = bt / 100

	dt := getValue(client, 0x3111)
	if dt > 32768 {
		dt = dt - 65536
	}
	c.DeviceTemp = dt / 100

	return
}

func getValue(client modbus.Client, address uint16) float32 {
	data, err := client.ReadInputRegisters(address, 1)
	if err != nil {
		log.Warnf("failed to get data, address: %d", address)
		return 0 // todo
	}
	return float32(binary.BigEndian.Uint16(data))
}

func getValue32(client modbus.Client, lowAddress uint16) float32 {
	lowData, err := client.ReadInputRegisters(lowAddress, 1)
	if err != nil {
		log.Warnf("failed to get data, address: %d", lowAddress)
		return 0 // todo
	}

	highAddress := lowAddress + 1

	highData, err := client.ReadInputRegisters(highAddress, 1)
	if err != nil {
		log.Warnf("failed to get data, address: %d", highAddress)
		return 0 // todo
	}

	swappedData := append(highData,lowData...)
	return float32(binary.BigEndian.Uint32(swappedData))
}