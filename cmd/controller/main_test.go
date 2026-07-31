package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lumberbarons/solar-controller/internal/config"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validConfig = `
solarController:
  httpPort: 9090
  deviceId: controller-test
  debug: false
  epever:
    enabled: true
    serialPort: /dev/ttyXRUSB0
    publishPeriod: 60
`

// writeConfig writes contents to a temp file and returns its path.
func writeConfig(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return path
}

func TestLoadConfig(t *testing.T) {
	t.Run("parses a valid config file", func(t *testing.T) {
		cfg, err := loadConfig(writeConfig(t, validConfig))
		require.NoError(t, err)

		assert.Equal(t, 9090, cfg.SolarController.HTTPPort)
		assert.Equal(t, "controller-test", cfg.SolarController.DeviceID)
		assert.True(t, cfg.SolarController.Epever.Enabled)
		assert.Equal(t, "/dev/ttyXRUSB0", cfg.SolarController.Epever.SerialPort)
	})

	t.Run("rejects an empty path instead of reading the working directory", func(t *testing.T) {
		_, err := loadConfig("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must specify config file path")
	})

	t.Run("reports a missing file", func(t *testing.T) {
		_, err := loadConfig(filepath.Join(t.TempDir(), "absent.yaml"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config file")
	})

	t.Run("reports malformed YAML", func(t *testing.T) {
		_, err := loadConfig(writeConfig(t, "solarController: [this is not a mapping"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load configuration")
	})

	t.Run("reports a config that parses but fails validation", func(t *testing.T) {
		_, err := loadConfig(writeConfig(t, "solarController:\n  httpPort: 0\n"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load configuration")
	})
}

func TestResolveDebugMode(t *testing.T) {
	tests := []struct {
		name      string
		debugFlag bool
		fileDebug bool
		want      bool
	}{
		{name: "off in both", debugFlag: false, fileDebug: false, want: false},
		{name: "flag enables it for a config that does not ask", debugFlag: true, fileDebug: false, want: true},
		{name: "config file alone enables it", debugFlag: false, fileDebug: true, want: true},
		{name: "on in both", debugFlag: true, fileDebug: true, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				SolarController: config.SolarControllerConfiguration{Debug: tt.fileDebug},
			}
			assert.Equal(t, tt.want, resolveDebugMode(tt.debugFlag, &cfg))
		})
	}
}

func TestConfigureLogging(t *testing.T) {
	original := log.GetLevel()
	t.Cleanup(func() { log.SetLevel(original) })

	configureLogging(true)
	assert.Equal(t, log.DebugLevel, log.GetLevel())

	configureLogging(false)
	assert.Equal(t, log.InfoLevel, log.GetLevel())
}

func TestVersionInfo(t *testing.T) {
	// Defaults stand in for the values ldflags injects at build time.
	info := versionInfo()

	assert.Equal(t, Version, info.Version)
	assert.Equal(t, BuildTime, info.BuildTime)
	assert.Equal(t, GitCommit, info.GitCommit)
}

func TestRunReturnsConfigErrorsRatherThanExiting(t *testing.T) {
	// run must surface a startup failure as an error; main is the only place
	// allowed to turn that into a process exit.
	err := run("", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must specify config file path")
}
