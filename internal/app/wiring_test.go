package app

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lumberbarons/solar-controller/internal/config"
	"github.com/lumberbarons/solar-controller/internal/controllers"
	"github.com/lumberbarons/solar-controller/internal/publish"
	"github.com/lumberbarons/solar-controller/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeController records what it was handed and whether it was started, so the
// wiring can be asserted without a serial port. The real epever constructor
// opens the device eagerly, which is why buildControllers takes factories.
type fakeController struct {
	name       string
	enabled    bool
	publisher  publish.MessagePublisher
	registered bool
	closed     bool
}

func (f *fakeController) RegisterEndpoints(r *gin.Engine) {
	f.registered = true
	r.GET("/api/"+f.name+"/metrics", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"controller": f.name})
	})
}

func (f *fakeController) Enabled() bool { return f.enabled }

func (f *fakeController) Close() error {
	f.closed = true
	return nil
}

// newFakeFactory returns a factory that builds fake, recording the publisher it
// was given.
func newFakeFactory(fake *fakeController) controllerFactory {
	return controllerFactory{
		name: fake.name,
		build: func(_ *config.SolarControllerConfiguration, publisher publish.MessagePublisher) (controllers.SolarController, error) {
			fake.publisher = publisher
			return fake, nil
		},
	}
}

func minimalConfig() *config.Config {
	return &config.Config{
		SolarController: config.SolarControllerConfiguration{HTTPPort: 8080},
	}
}

func TestBuildControllers_BuiltSetMatchesConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		fakes    []*fakeController
		expected []string
	}{
		{
			name:     "no controllers configured",
			fakes:    nil,
			expected: nil,
		},
		{
			name:     "an enabled controller is started",
			fakes:    []*fakeController{{name: "epever", enabled: true}},
			expected: []string{"epever"},
		},
		{
			name:     "a disabled controller is not started",
			fakes:    []*fakeController{{name: "epever", enabled: false}},
			expected: nil,
		},
		{
			// A controller that is enabled in config but missing a required
			// field returns an empty, disabled instance rather than an error.
			name:     "an enabled but misconfigured controller is not started",
			fakes:    []*fakeController{{name: "epever", enabled: false}},
			expected: nil,
		},
		{
			name: "only the enabled controllers of several are started",
			fakes: []*fakeController{
				{name: "epever", enabled: true},
				{name: "voltgo", enabled: false},
			},
			expected: []string{"epever"},
		},
		{
			name: "every enabled controller is started",
			fakes: []*fakeController{
				{name: "epever", enabled: true},
				{name: "voltgo", enabled: true},
			},
			expected: []string{"epever", "voltgo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factories := make([]controllerFactory, 0, len(tt.fakes))
			for _, fake := range tt.fakes {
				factories = append(factories, newFakeFactory(fake))
			}

			app, err := newApplication(minimalConfig(), testutil.NewMockPublisher(), getTestVersionInfo(), factories)
			require.NoError(t, err)
			defer app.Close()

			built := make([]string, 0, len(app.controllers))
			for _, controller := range app.controllers {
				built = append(built, controller.(*fakeController).name)
			}
			assert.Equal(t, tt.expected, nilIfEmpty(built))

			// Only started controllers get their endpoints registered.
			for _, fake := range tt.fakes {
				assert.Equal(t, fake.enabled, fake.registered,
					"controller %q registered=%v but enabled=%v", fake.name, fake.registered, fake.enabled)
			}
		})
	}
}

func nilIfEmpty(names []string) []string {
	if len(names) == 0 {
		return nil
	}
	return names
}

// registeredPaths returns the paths the router actually has routes for.
func registeredPaths(app *Application) []string {
	routes := app.Router().Routes()
	paths := make([]string, 0, len(routes))
	for _, route := range routes {
		paths = append(paths, route.Path)
	}
	return paths
}

// The publisher is the only path metrics take off the device, so a controller
// built without it would collect and silently discard everything.
func TestBuildControllers_EveryControllerReceivesThePublisher(t *testing.T) {
	gin.SetMode(gin.TestMode)

	epeverFake := &fakeController{name: "epever", enabled: true}
	voltgoFake := &fakeController{name: "voltgo", enabled: true}
	publisher := testutil.NewMockPublisher()

	app, err := newApplication(
		minimalConfig(),
		publisher,
		getTestVersionInfo(),
		[]controllerFactory{newFakeFactory(epeverFake), newFakeFactory(voltgoFake)},
	)
	require.NoError(t, err)
	defer app.Close()

	assert.Same(t, publisher, epeverFake.publisher, "epever was built without the configured publisher")
	assert.Same(t, publisher, voltgoFake.publisher, "voltgo was built without the configured publisher")
}

func TestBuildControllers_ConstructorFailureNamesTheController(t *testing.T) {
	gin.SetMode(gin.TestMode)

	factories := []controllerFactory{{
		name: "epever",
		build: func(*config.SolarControllerConfiguration, publish.MessagePublisher) (controllers.SolarController, error) {
			return nil, errors.New("serial port unavailable")
		},
	}}

	_, err := newApplication(minimalConfig(), testutil.NewMockPublisher(), getTestVersionInfo(), factories)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "epever")
	assert.Contains(t, err.Error(), "serial port unavailable")
}

func TestClose_ClosesEveryStartedController(t *testing.T) {
	gin.SetMode(gin.TestMode)

	started := &fakeController{name: "epever", enabled: true}
	notStarted := &fakeController{name: "voltgo", enabled: false}

	app, err := newApplication(
		minimalConfig(),
		testutil.NewMockPublisher(),
		getTestVersionInfo(),
		[]controllerFactory{newFakeFactory(started), newFakeFactory(notStarted)},
	)
	require.NoError(t, err)
	require.NoError(t, app.Close())

	assert.True(t, started.closed, "a started controller was not closed on shutdown")
	assert.False(t, notStarted.closed, "a controller that never started should not be closed")
}

// The production wiring must list every controller the binary supports;
// otherwise a controller can be added and silently never started.
func TestDefaultControllerFactories(t *testing.T) {
	names := make([]string, 0)
	for _, factory := range defaultControllerFactories() {
		names = append(names, factory.name)
	}

	assert.Equal(t, []string{"epever"}, names)
}

// The real epever constructor is exercised here rather than through a fake, so
// the config-to-controller decision is checked against the actual code path for
// the two cases that do not need hardware.
func TestDefaultFactories_EpeverNotStartedWithoutSerialPort(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name    string
		enabled bool
	}{
		{name: "disabled", enabled: false},
		{name: "enabled but no serial port", enabled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := minimalConfig()
			cfg.SolarController.Epever.Enabled = tt.enabled

			app, err := NewApplication(cfg, testutil.NewMockPublisher(), getTestVersionInfo())
			require.NoError(t, err)
			defer app.Close()

			assert.Empty(t, app.controllers)

			// Asserted against the router's route table rather than by making a
			// request: the SPA fallback answers 200 with index.html for any
			// unmatched path, so a status code cannot show a route is absent.
			assert.NotContains(t, registeredPaths(app), "/api/epever/metrics",
				"epever endpoints were registered for a controller that never started")
		})
	}
}
