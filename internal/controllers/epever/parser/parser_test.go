package parser

import (
	"encoding/binary"
	"testing"
)

func TestParseFloat(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    float32
		wantErr bool
	}{
		{
			name: "valid voltage 12.8V",
			data: []byte{0x05, 0x00}, // 1280 in big-endian
			want: 12.8,
		},
		{
			name: "valid voltage 18.5V",
			data: []byte{0x07, 0x39}, // 1849 in big-endian
			want: 18.49,
		},
		{
			name: "zero value",
			data: []byte{0x00, 0x00},
			want: 0.0,
		},
		{
			name:    "insufficient data",
			data:    []byte{0x05},
			wantErr: true,
		},
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFloat(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFloat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !floatEqual(got, tt.want) {
				t.Errorf("ParseFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseFloats(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		quantity int
		want     []float32
		wantErr  bool
	}{
		{
			name:     "two values (voltage and current)",
			data:     []byte{0x05, 0x00, 0x02, 0x08}, // 1280, 520
			quantity: 2,
			want:     []float32{12.8, 5.2},
		},
		{
			name:     "single value",
			data:     []byte{0x07, 0x39}, // 1849
			quantity: 1,
			want:     []float32{18.49},
		},
		{
			name:     "three values",
			data:     []byte{0x05, 0x00, 0x02, 0x08, 0x07, 0x39},
			quantity: 3,
			want:     []float32{12.8, 5.2, 18.49},
		},
		{
			name:     "insufficient data",
			data:     []byte{0x05, 0x00, 0x02}, // Only 3 bytes for 2 values
			quantity: 2,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFloats(tt.data, tt.quantity)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFloats() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("ParseFloats() length = %d, want %d", len(got), len(tt.want))
					return
				}
				for i := range got {
					if !floatEqual(got[i], tt.want[i]) {
						t.Errorf("ParseFloats()[%d] = %v, want %v", i, got[i], tt.want[i])
					}
				}
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    int32
		wantErr bool
	}{
		{
			name: "battery SOC 85%",
			data: []byte{0x00, 0x55}, // 85
			want: 85,
		},
		{
			name: "zero value",
			data: []byte{0x00, 0x00},
			want: 0,
		},
		{
			name: "max uint16",
			data: []byte{0xFF, 0xFF},
			want: 65535,
		},
		{
			name:    "insufficient data",
			data:    []byte{0x05},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInt(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseInts(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		quantity int
		want     []int32
		wantErr  bool
	}{
		{
			name:     "two values",
			data:     []byte{0x00, 0x55, 0x00, 0x64}, // 85, 100
			quantity: 2,
			want:     []int32{85, 100},
		},
		{
			name:     "temperature registers",
			data:     []byte{0x09, 0xC4, 0x0C, 0x80}, // 2500, 3200
			quantity: 2,
			want:     []int32{2500, 3200},
		},
		{
			name:     "insufficient data",
			data:     []byte{0x00, 0x55, 0x00}, // Only 3 bytes for 2 values
			quantity: 2,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInts(tt.data, tt.quantity)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("ParseInts() length = %d, want %d", len(got), len(tt.want))
					return
				}
				for i := range got {
					if got[i] != tt.want[i] {
						t.Errorf("ParseInts()[%d] = %v, want %v", i, got[i], tt.want[i])
					}
				}
			}
		})
	}
}

func TestParseFloat32(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    float32
		wantErr bool
	}{
		{
			name: "energy value 15.50 kWh",
			data: createFloat32Data(1550), // 1550 centi-kWh = 15.50 kWh
			want: 15.50,
		},
		{
			name: "zero value",
			data: createFloat32Data(0),
			want: 0.0,
		},
		{
			name: "large energy value",
			data: createFloat32Data(120000), // 1200.00 kWh
			want: 1200.00,
		},
		{
			name:    "insufficient data",
			data:    []byte{0x00, 0x00, 0x00},
			wantErr: true,
		},
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFloat32(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFloat32() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !floatEqual(got, tt.want) {
				t.Errorf("ParseFloat32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseSignedTemperature(t *testing.T) {
	tests := []struct {
		name string
		raw  int32
		want float32
	}{
		{
			name: "positive temperature 25°C",
			raw:  2500, // 2500 centi-degrees
			want: 25.0,
		},
		{
			name: "zero temperature",
			raw:  0,
			want: 0.0,
		},
		{
			name: "negative temperature -10°C",
			raw:  64536, // 65536 - 1000 (represented as unsigned)
			want: -10.0,
		},
		{
			name: "negative temperature -5°C",
			raw:  65036, // 65536 - 500
			want: -5.0,
		},
		{
			name: "high positive temperature 45°C",
			raw:  4500,
			want: 45.0,
		},
		{
			name: "threshold boundary (just below)",
			raw:  32767,
			want: 327.67,
		},
		{
			name: "threshold boundary (just above)",
			raw:  32768, // Should be treated as negative
			want: -327.68,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSignedTemperature(tt.raw)
			if !floatEqual(got, tt.want) {
				t.Errorf("ParseSignedTemperature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseTemperatures(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		wantBattery float32
		wantDevice  float32
		wantErr     bool
	}{
		{
			name:        "positive temperatures",
			data:        []byte{0x09, 0xC4, 0x0C, 0x80}, // 2500, 3200
			wantBattery: 25.0,
			wantDevice:  32.0,
		},
		{
			name:        "negative battery temp",
			data:        []byte{0xFC, 0x18, 0x0C, 0x80}, // 64536 (-10°C), 3200 (32°C)
			wantBattery: -10.0,
			wantDevice:  32.0,
		},
		{
			name:        "both negative",
			data:        []byte{0xFE, 0x0C, 0xFC, 0x18}, // 65036 (-5°C), 64536 (-10°C)
			wantBattery: -5.0,
			wantDevice:  -10.0,
		},
		{
			name:    "insufficient data",
			data:    []byte{0x09, 0xC4, 0x0C}, // Only 3 bytes
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBattery, gotDevice, err := ParseTemperatures(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTemperatures() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !floatEqual(gotBattery, tt.wantBattery) {
					t.Errorf("ParseTemperatures() battery = %v, want %v", gotBattery, tt.wantBattery)
				}
				if !floatEqual(gotDevice, tt.wantDevice) {
					t.Errorf("ParseTemperatures() device = %v, want %v", gotDevice, tt.wantDevice)
				}
			}
		})
	}
}

func TestEncodeUint16(t *testing.T) {
	tests := []struct {
		name  string
		value uint16
		want  []byte
	}{
		{
			name:  "voltage 1280 (12.8V)",
			value: 1280,
			want:  []byte{0x05, 0x00},
		},
		{
			name:  "zero",
			value: 0,
			want:  []byte{0x00, 0x00},
		},
		{
			name:  "max uint16",
			value: 65535,
			want:  []byte{0xFF, 0xFF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeUint16(tt.value)
			if len(got) != len(tt.want) {
				t.Errorf("EncodeUint16() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("EncodeUint16()[%d] = %02X, want %02X", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestEncodeUint16s(t *testing.T) {
	tests := []struct {
		name   string
		values []uint16
		want   []byte
	}{
		{
			name:   "two values",
			values: []uint16{1280, 520},
			want:   []byte{0x05, 0x00, 0x02, 0x08},
		},
		{
			name:   "single value",
			values: []uint16{1849},
			want:   []byte{0x07, 0x39},
		},
		{
			name:   "empty slice",
			values: []uint16{},
			want:   []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeUint16s(tt.values)
			if len(got) != len(tt.want) {
				t.Errorf("EncodeUint16s() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("EncodeUint16s()[%d] = %02X, want %02X", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestEncodeVoltage(t *testing.T) {
	tests := []struct {
		name    string
		voltage float32
		want    uint16
	}{
		{
			name:    "12.8V",
			voltage: 12.8,
			want:    1280,
		},
		{
			name:    "18.5V",
			voltage: 18.5,
			want:    1850,
		},
		{
			name:    "zero",
			voltage: 0.0,
			want:    0,
		},
		{
			name:    "14.4V (boost voltage)",
			voltage: 14.4,
			want:    1440,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeVoltage(tt.voltage)
			if got != tt.want {
				t.Errorf("EncodeVoltage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEncodeTemperature(t *testing.T) {
	tests := []struct {
		name string
		temp float32
		want int16
	}{
		{
			name: "25°C",
			temp: 25.0,
			want: 2500,
		},
		{
			name: "0°C",
			temp: 0.0,
			want: 0,
		},
		{
			name: "-10°C",
			temp: -10.0,
			want: -1000,
		},
		{
			name: "45°C",
			temp: 45.0,
			want: 4500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeTemperature(tt.temp)
			if got != tt.want {
				t.Errorf("EncodeTemperature() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper functions

// floatEqual checks if two float32 values are approximately equal
func floatEqual(a, b float32) bool {
	tolerance := float32(0.01)
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < tolerance
}

// createFloat32Data creates byte data for a float32 value with Epever byte swapping
func createFloat32Data(value uint32) []byte {
	data := make([]byte, 4)
	// Epever stores low word first, then high word
	binary.BigEndian.PutUint16(data[2:4], uint16(value>>16))
	binary.BigEndian.PutUint16(data[0:2], uint16(value&0xFFFF))
	return data
}
