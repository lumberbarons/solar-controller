package pijuice

import (
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2creg"
	"strconv"
	"time"
)

type Collector struct {
	Bus     string
	Address uint64
}

type ControllerStatus struct {
	Timestamp int64 `json:"timestamp"`

	BatteryStatus    int32 `json:"batteryStatus"`
	PowerInputStatus int32 `json:"powerInputStatus"`
}

func NewCollector(bus string, address string) (*Collector, error) {
	decodedAddress, err := strconv.ParseUint(address, 0, 10)
	if err != nil {
		return nil, err
	}

	collector := &Collector{
		Bus: bus,
		Address: decodedAddress,
	}

	return collector, nil
}

func (e *Collector) GetStatus() (*ControllerStatus, error) {
	startTime := time.Now()

	c := &ControllerStatus{
		Timestamp: startTime.Unix(),
	}

	bus, err := i2creg.Open(e.Bus)
	if err != nil {
		return nil, err
	}
	defer bus.Close()

	d := i2c.Dev{Bus: bus, Addr: uint16(e.Address)}

	write := []byte{0x40}
	status := make([]byte, 1)

	err = d.Tx(write, status)
	if err != nil {
		return nil, err
	}

	return c, nil
}