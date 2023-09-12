package victron

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/go-co-op/gocron"
	"github.com/rigado/ble"
	"github.com/rigado/ble/linux"
	"path/filepath"
	"strings"
	"time"

	bonds "github.com/rigado/ble/linux/hci/bond"
	log "github.com/sirupsen/logrus"
)

type Collector struct {
	client        ble.Client
	service       *ble.Service
	keepaliveChar *ble.Characteristic
	metricChars   []*ble.Characteristic
	done          chan struct{}
}

type ControllerStatus struct {
	Timestamp      int64     `json:"timestamp"`
	CollectionTime float64   `json:"collectionTime"`
	ConsumedAh     float32   `json:"consumedAh"`
	Power          int16     `json:"power"`
	Voltage        float32   `json:"voltage"`
	Current		   float32   `json:"current"`
	StateOfCharge  float32   `json:"stateOfCharge"`
}

var consumedAhUuid    ble.UUID
var powerUuid         ble.UUID
var voltageUuid       ble.UUID
var currentUuid       ble.UUID
var stateOfChargeUuid ble.UUID
var keepaliveUuid     ble.UUID

func NewCollector(config Configuration, scheduler *gocron.Scheduler) (*Collector, error) {
	consumedAhUuid, _ = ble.Parse("6597eeff-4bda-4c1e-af4b-551c4cf74769")
	powerUuid, _ = ble.Parse("6597ed8e-4bda-4c1e-af4b-551c4cf74769")
	voltageUuid, _ = ble.Parse("6597ed8d-4bda-4c1e-af4b-551c4cf74769")
	currentUuid, _ = ble.Parse("6597ed8c-4bda-4c1e-af4b-551c4cf74769")
	stateOfChargeUuid, _ = ble.Parse("65970fff-4bda-4c1e-af4b-551c4cf74769")
	keepaliveUuid, _ = ble.Parse("6597ffff-4bda-4c1e-af4b-551c4cf74769")

	victronClient, err := connect(config)
	if err != nil {
		if victronClient != nil {
			victronClient.disconnect()
		}
		return nil, fmt.Errorf("failed to connect to device: %w", err)
	}

	_, err = scheduler.Every(30).Seconds().WaitForSchedule().Do(victronClient.writeKeepalive)
	if err != nil {
		victronClient.disconnect()
		return nil, fmt.Errorf("failed to start victron keepalive: %w", err)
	}

	return victronClient, nil
}

func connect(config Configuration) (*Collector, error) {
	log.Printf("connecting to device %s", config.MacAddress)

	optid := ble.OptDeviceID(0)
	bondFilePath := filepath.Join("bonds.json")
	bm := bonds.NewBondManager(bondFilePath)

	optSecurity := ble.OptEnableSecurity(bm)
	d, err := linux.NewDeviceWithNameAndHandler("", nil, optid, optSecurity)
	if err != nil {
		return nil, fmt.Errorf("can't create new device: %w", err)
	}

	ble.SetDefaultDevice(d)

	filter := func(a ble.Advertisement) bool {
		return strings.ToUpper(a.Addr().String()) == strings.ToUpper(config.MacAddress)
	}

	scanDuration := 20 * time.Second

	log.Printf("Scanning for %s...\n", scanDuration)
	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), scanDuration))
	client, err := ble.Connect(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	done := make(chan struct{})

	go func() {
		<- client.Disconnected()
		log.Println("disconnected")
		close(done)
	}()

	log.Println("connected, pairing with", client.Addr().String())

	ad := ble.AuthData{}
	ad.Passkey = 0

	victronClient := &Collector{client: client, done: done}

	err = client.Pair(ad, time.Minute)
	if err != nil {
		return victronClient, fmt.Errorf("failed to pair: %w", err)
	}

	log.Println("pairing successful")

	serviceUUID, _ := ble.Parse("65970000-4bda-4c1e-af4b-551c4cf74769")

	services, err := client.DiscoverServices([]ble.UUID{serviceUUID})
	if err != nil {
		return nil, fmt.Errorf("failed to discover services: %w", err)
	}

	if len(services) == 0 {
		return nil, fmt.Errorf("failed to discover services")
	}

	var service *ble.Service
	for _, s := range services {
		if serviceUUID.Equal(s.UUID) {
			service = s
			break
		}
	}

	if service == nil {
		return nil, fmt.Errorf("failed to discover service")
	}

	victronClient.service = service

	metricCharsFilter := []ble.UUID{consumedAhUuid, powerUuid, voltageUuid, currentUuid, stateOfChargeUuid}
	metricChars, err := client.DiscoverCharacteristics(metricCharsFilter, service)
	if err != nil {
		return nil, fmt.Errorf("failed to discover metric characteristics: %w", err)
	}

	if len(metricChars) == 0 {
		return nil, fmt.Errorf("failed to discover metric characteristics")
	}

	victronClient.metricChars = metricChars

	keepaliveCharFilter := []ble.UUID{keepaliveUuid}
	keepaliveChars, err := client.DiscoverCharacteristics(keepaliveCharFilter, service)
	if err != nil {
		return nil, fmt.Errorf("failed to discover metric characteristics: %w", err)
	}

	if len(keepaliveChars) == 0 {
		return nil, fmt.Errorf("failed to discover keepalive characteristic")
	}

	var keepaliveChar *ble.Characteristic
	for _, c := range keepaliveChars {
		if keepaliveUuid.Equal(c.UUID) {
			keepaliveChar = c
			break
		}
	}

	if keepaliveChar == nil {
		return nil, fmt.Errorf("failed to discover keepalive characteristic")
	}

	victronClient.keepaliveChar = keepaliveChar

	return victronClient, nil
}

func (v *Collector) disconnect() {
	log.Println("victron disconnecting...")

	err := v.client.CancelConnection()
	if err != nil {
		log.Println("failed to cancel connection to victron:", err)
	}

	<- v.done
}

func (v *Collector) writeKeepalive() {
	log.Println("victron keepalive start")

	keepalive, _ := hex.DecodeString("60ea") // 60 seconds

	err := v.client.WriteCharacteristic(v.keepaliveChar, keepalive, false)
	if err != nil {
		log.Printf("failed to write keepalive: %s", err)
	}

	log.Println("victron keepalive done")
}

func (v *Collector) GetStatus() (*ControllerStatus, error) {
	startTime := time.Now()

	status := &ControllerStatus{
		Timestamp: startTime.Unix(),
	}

	for _, c := range v.metricChars {
		value, err := v.client.ReadCharacteristic(c)
		if err != nil {
			return nil, fmt.Errorf("failed to read value from victron: %w", err)
		}

		if consumedAhUuid.Equal(c.UUID) {
			status.ConsumedAh = float32(readInt32(value)) * 0.1
		} else if powerUuid.Equal(c.UUID) {
			status.Power = readInt16(value)
		} else if voltageUuid.Equal(c.UUID) {
			status.Voltage = float32(readInt16(value)) * 0.01
		} else if currentUuid.Equal(c.UUID) {
			status.Current = float32(readInt16(value)) * 0.001
		} else if stateOfChargeUuid.Equal(c.UUID) {
			status.StateOfCharge = float32(readUnsignedInt16(value)) * 0.01
		}
	}

	status.CollectionTime = time.Now().Sub(startTime).Seconds()

	return status, nil
}

func readInt32(data []byte) (resp int32) {
	buf := bytes.NewBuffer(data)
	binary.Read(buf, binary.LittleEndian, &resp)
	return resp
}

func readInt16(data []byte) (resp int16) {
	buf := bytes.NewBuffer(data)
	binary.Read(buf, binary.LittleEndian, &resp)
	return resp
}

func readUnsignedInt16(data []byte) (resp uint16) {
	buf := bytes.NewBuffer(data)
	binary.Read(buf, binary.LittleEndian, &resp)
	return resp
}
