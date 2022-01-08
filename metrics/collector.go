package metrics

import (
	"encoding/binary"
	"github.com/goburrow/serial"
	log "github.com/sirupsen/logrus"
	"time"
)

type ControllerStatus struct {
	ArrayVoltage           float32   `json:"arrayVoltage"`
	ArrayCurrent           float32   `json:"arrayCurrent"`
	ArrayPower             float32   `json:"arrayPower"`
	BatteryVoltage         float32   `json:"batteryVoltage"`
	BatterySOC             float32   `json:"batterySoc"`
	BatteryTemp            float32   `json:"batteryTemp"`
	BatteryMaxVoltage      float32   `json:"batteryMaxVoltage"`
	BatteryMinVoltage      float32   `json:"batteryMinVoltage"`
	DeviceTemp             float32   `json:"deviceTemp"`
	EnergyGeneratedDaily   float32   `json:"energyGeneratedDaily"`
	EnergyGeneratedMonthly float32   `json:"energyGeneratedMonthly"`
	EnergyGeneratedAnnual  float32   `json:"energyGeneratedAnnually"`
	EnergyGeneratedTotal   float32   `json:"energyGeneratedTotal"`
	Timestamp              time.Time `json:"timestamp"`
}

func getStatus(portName string) (c ControllerStatus, err error) {
	config := serial.Config{
		Address:  portName,
		BaudRate: 115200,
		DataBits: 8,
		StopBits: 1,
		Parity:   "N",
		Timeout:  5 * time.Second,
	}

	port, err := serial.Open(&config)
	if err != nil {
		return
	}

	log.Debugln("serial connected")

	defer func() {
		err := port.Close()
		if err != nil {
			log.Error("failed to close serial port: %w", err)
		}
		log.Debugln("serial closed")
	}()

	c.Timestamp = time.Now().UTC()

	c.ArrayVoltage = getValue(port, 0x3100) / 100
	c.ArrayCurrent = getValue(port, 0x3101) / 100
	c.BatteryVoltage = getValue(port, 0x3104) / 100
	c.BatterySOC = getValue(port, 0x311A)

	c.BatteryMaxVoltage = getValue(port, 0x3302) / 100
	c.BatteryMinVoltage = getValue(port, 0x3303) / 100

	c.ArrayPower = getValue32(port, 0x3102) / 100

	c.EnergyGeneratedDaily = getValue32(port, 0x330C) / 100
	c.EnergyGeneratedMonthly = getValue32(port, 0x330E) / 100
	c.EnergyGeneratedAnnual = getValue32(port, 0x3310) / 100
	c.EnergyGeneratedTotal = getValue32(port, 0x3312) / 100

	bt := getValue(port, 0x3110)
	if bt > 32768 {
		bt = bt - 65536
	}
	c.BatteryTemp = bt / 100

	dt := getValue(port, 0x3111)
	if dt > 32768 {
		dt = dt - 65536
	}
	c.DeviceTemp = dt / 100

	return
}

func getValue(port serial.Port, address uint16) float32 {
	data,err := getData(port, address)
	if err != nil {
		log.Warn("failed to get data, address:", address)
		return 0 // todo
	}
	return float32(binary.BigEndian.Uint16(data))
}

func getValue32(port serial.Port, lowAddress uint16) float32 {
	lowData,err := getData(port, lowAddress)
	if err != nil {
		log.Warn("failed to get data, address:", lowAddress)
		return 0 // todo
	}

	highAddress := lowAddress + 1

	highData, err := getData(port, highAddress)
	if err != nil {
		log.Warn("failed to get data, address: %d", highAddress)
		return 0 // todo
	}

	swappedData := append(highData,lowData...)
	return float32(binary.BigEndian.Uint32(swappedData))
}

func getData(port serial.Port, address uint16) ([]byte, error) {
	addressBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(addressBytes, address)

	data := append([]byte{0x01, 0x04}, addressBytes...)
	data = append(data, 0x00, 0x01)

	crc := make([]byte, 2)
	binary.LittleEndian.PutUint16(crc, calculateCrc(data))
	data = append(data, crc...)

	//log.Printf("request: %x\n", data)

	if _, err := port.Write(data); err != nil {
		return nil, err
	}

	response := make([]byte, 7)
	if _, err := port.Read(response); err != nil {
		return nil, err
	}

	//log.Printf("data: %x\n", response[3:5])

	time.Sleep(100 * time.Millisecond)
	return response[3:5], nil
}

func calculateCrc(data []byte) uint16 {
	var crc16 uint16 = 0xffff
	for i := 0; i < len(data); i++ {
		crc16 ^= uint16(data[i])
		for j := 0; j < 8; j++ {
			if crc16&0x0001 > 0 {
				crc16 = (crc16 >> 1) ^ 0xA001
			} else {
				crc16 >>= 1
			}
		}
	}
	return crc16
}
