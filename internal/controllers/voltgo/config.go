package voltgo

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

const defaultConnectTimeout = 30 * time.Second

// batteryIDPattern constrains a battery id to characters that are safe
// unescaped in all three places an id appears: a URL path segment, a
// Prometheus label value, and an MQTT/Solace topic segment. Slashes, spaces,
// and `+`/`#` would each break at least one of those.
var batteryIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// BatteryConfiguration describes one physical battery on the BLE bus.
type BatteryConfiguration struct {
	// ID identifies this battery in routes, Prometheus labels, and publisher
	// topics. Optional: defaults to the address, lowercased with colons
	// replaced by dashes.
	ID string `yaml:"id"`

	// Address is the BLE address of the battery, e.g. AA:BB:CC:DD:EE:FF.
	Address string `yaml:"address"`
}

type Configuration struct {
	Enabled bool `yaml:"enabled"`

	// Address configures a single battery.
	//
	// Deprecated: use Batteries. Retained so an existing single-battery
	// config keeps working; it is an error to set both.
	Address string `yaml:"address"`

	// Batteries lists every battery to collect from. Collection is
	// serialised across them, so PublishPeriod must leave room for one
	// connect-and-read per battery.
	Batteries []BatteryConfiguration `yaml:"batteries"`

	PublishPeriod int `yaml:"publishPeriod"`

	// ConnectTimeout is the maximum time to wait for a BLE connection,
	// as a duration string (default: 30s). It applies per battery.
	ConnectTimeout string `yaml:"connectTimeout"`
}

// Validate checks the configuration for errors. Only called when enabled.
func (c *Configuration) Validate() error {
	if c.ConnectTimeout != "" {
		if _, err := time.ParseDuration(c.ConnectTimeout); err != nil {
			return err
		}
	}
	if _, err := c.ResolveBatteries(); err != nil {
		return err
	}
	return nil
}

// ResolveBatteries normalises the singular and list forms into one list with
// every id filled in, and rejects a configuration that cannot be collected
// from unambiguously.
//
// Duplicate ids and duplicate addresses are both errors rather than warnings:
// a duplicate id would silently collapse two batteries onto one route,
// one Prometheus series, and one topic, and a duplicate address would have two
// runners fighting over the same BLE connection.
func (c *Configuration) ResolveBatteries() ([]BatteryConfiguration, error) {
	if c.Address != "" && len(c.Batteries) > 0 {
		return nil, fmt.Errorf("voltgo address and batteries are mutually exclusive; " +
			"move the single address into the batteries list")
	}

	configured := c.Batteries
	if c.Address != "" {
		configured = []BatteryConfiguration{{Address: c.Address}}
	}

	resolved := make([]BatteryConfiguration, 0, len(configured))
	seenIDs := make(map[string]struct{}, len(configured))
	seenAddrs := make(map[string]struct{}, len(configured))

	for i, battery := range configured {
		if battery.Address == "" {
			return nil, fmt.Errorf("voltgo battery %d: address is required", i)
		}

		id := battery.ID
		if id == "" {
			id = DefaultBatteryID(battery.Address)
		}
		if !batteryIDPattern.MatchString(id) {
			return nil, fmt.Errorf("voltgo battery %q: id must match %s", id, batteryIDPattern)
		}

		if _, dup := seenIDs[id]; dup {
			return nil, fmt.Errorf("voltgo battery id %q is configured more than once", id)
		}
		seenIDs[id] = struct{}{}

		// Addresses are compared case-insensitively: BlueZ and the
		// datasheet disagree on the case of the hex digits, so the same
		// battery is easily written both ways.
		addr := strings.ToUpper(battery.Address)
		if _, dup := seenAddrs[addr]; dup {
			return nil, fmt.Errorf("voltgo battery address %q is configured more than once", battery.Address)
		}
		seenAddrs[addr] = struct{}{}

		resolved = append(resolved, BatteryConfiguration{ID: id, Address: battery.Address})
	}

	return resolved, nil
}

// DefaultBatteryID derives an id from a BLE address, for a battery configured
// without one.
func DefaultBatteryID(address string) string {
	return strings.ToLower(strings.ReplaceAll(address, ":", "-"))
}

// GetConnectTimeout returns the configured connect timeout or the default (30s).
func (c *Configuration) GetConnectTimeout() time.Duration {
	if c.ConnectTimeout == "" {
		return defaultConnectTimeout
	}
	timeout, err := time.ParseDuration(c.ConnectTimeout)
	if err != nil {
		return defaultConnectTimeout
	}
	return timeout
}
