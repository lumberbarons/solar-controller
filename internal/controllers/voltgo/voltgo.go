package voltgo

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"github.com/lumberbarons/solar-controller/internal/publish"
	log "github.com/sirupsen/logrus"
)

const (
	namespace = "voltgo"

	// collectTimeout bounds one battery's collection: a BLE connect
	// (default 30s) plus the status reads. Each battery in a cycle gets its
	// own budget, because a cycle is serialised across them.
	collectTimeout = 60 * time.Second
)

// BatteryRef identifies one configured battery in the index endpoint, so the
// frontend can discover which batteries exist rather than hardcoding them.
type BatteryRef struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

// BatteryIndex is the payload of GET /api/voltgo.
type BatteryIndex struct {
	Batteries []BatteryRef `json:"batteries"`
}

// Controller drives every configured voltgo battery. It is one controller
// rather than one per battery because the three things that have to be unique
// - the Gin routes, the Prometheus registration, and the BLE adapter - are all
// process-wide: registering them once here is what lets a second battery exist
// at all.
type Controller struct {
	batteries []*batteryRunner
	byID      map[string]*batteryRunner

	// connector is the shared BLE adapter behind every battery's collector,
	// owned here so it is released exactly once on Close.
	connector BatteryConnector

	scheduler *gocron.Scheduler

	collectInProgress bool
	collectMutex      sync.Mutex
}

// NewController creates a voltgo controller from already-built battery
// runners, for testing. For production use, call NewControllerFromConfig.
func NewController(
	batteries []*batteryRunner,
	connector BatteryConnector,
	publishPeriod int,
) (*Controller, error) {
	if len(batteries) == 0 {
		return &Controller{}, nil
	}

	controller := newControllerForTest(batteries, connector)

	s := gocron.NewScheduler(time.UTC)
	controller.scheduler = s

	_, err := s.Every(publishPeriod).Seconds().Do(controller.collectAndPublish)
	if err != nil {
		return nil, fmt.Errorf("failed to start voltgo publisher %w", err)
	}

	s.StartAsync()

	// Run initial collection immediately
	go controller.collectAndPublish()

	return controller, nil
}

// newControllerForTest creates a Controller without starting the scheduler or
// background goroutine, so tests can call collectAndPublish synchronously
// without racing.
func newControllerForTest(batteries []*batteryRunner, connector BatteryConnector) *Controller {
	byID := make(map[string]*batteryRunner, len(batteries))
	for _, battery := range batteries {
		byID[battery.id] = battery
	}
	return &Controller{
		batteries: batteries,
		byID:      byID,
		connector: connector,
	}
}

// NewControllerFromConfig creates a new voltgo controller from configuration.
// This is the production entry point that creates all concrete dependencies.
func NewControllerFromConfig(config Configuration, publisher publish.MessagePublisher, deviceID string) (*Controller, error) {
	if !config.Enabled {
		log.Info("voltgo disabled via configuration")
		return &Controller{}, nil
	}

	batteryConfigs, err := config.ResolveBatteries()
	if err != nil {
		log.Warnf("voltgo enabled but not usable: %s", err)
		return &Controller{}, nil
	}
	if len(batteryConfigs) == 0 {
		log.Warn("voltgo enabled but no batteries configured")
		return &Controller{}, nil
	}

	if deviceID == "" {
		deviceID = "controller-1"
	}

	// One adapter is shared by every battery. BLE hardware exposes a single
	// adapter per host, and the collectors serialise their use of it.
	connector, err := NewBLEConnector()
	if err != nil {
		return nil, err
	}

	prometheusCollector := NewPrometheusCollector()
	connectTimeout := config.GetConnectTimeout()

	runners := make([]*batteryRunner, 0, len(batteryConfigs))
	for _, batteryConfig := range batteryConfigs {
		collector := NewCollector(connector, batteryConfig.Address, connectTimeout)
		runners = append(runners, newBatteryRunner(
			batteryConfig.ID, batteryConfig.Address, collector, publisher, prometheusCollector, deviceID,
		))
		log.Infof("voltgo battery %q configured at %s", batteryConfig.ID, batteryConfig.Address)
	}

	return NewController(runners, connector, config.PublishPeriod)
}

// collectAndPublish runs one collection cycle across every battery.
//
// Batteries are collected one after another rather than concurrently: the
// host has a single BLE adapter, BlueZ limits concurrent links, and the BMS is
// timing-sensitive enough that overlapping connects provoke the connection
// aborts that voltgo v0.2.1 fixed. The cost is that the publish period has to
// accommodate the sum of the per-battery connect-and-read times.
func (v *Controller) collectAndPublish() {
	// Check if a collection is already in progress
	v.collectMutex.Lock()
	if v.collectInProgress {
		log.Warn("collection already in progress for voltgo controller, skipping this collection cycle")
		v.collectMutex.Unlock()
		return
	}
	v.collectInProgress = true
	v.collectMutex.Unlock()

	// Ensure we clear the flag when done
	defer func() {
		v.collectMutex.Lock()
		v.collectInProgress = false
		v.collectMutex.Unlock()
	}()

	for _, battery := range v.batteries {
		// Each battery gets its own timeout so one unreachable battery
		// burns its own budget and not the whole cycle's.
		ctx, cancel := context.WithTimeout(context.Background(), collectTimeout)
		battery.collectAndPublish(ctx)
		cancel()
	}
}

// IndexGet lists the configured batteries. It answers 200 with a possibly
// empty list as soon as the controller is running, so the frontend can tell
// "voltgo is configured, batteries are still connecting" from the 404 that
// means no voltgo controller exists at all.
func (v *Controller) IndexGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		refs := make([]BatteryRef, 0, len(v.batteries))
		for _, battery := range v.batteries {
			refs = append(refs, BatteryRef{ID: battery.id, Address: battery.address})
		}
		c.JSON(http.StatusOK, BatteryIndex{Batteries: refs})
	}
}

func (v *Controller) MetricsGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		battery, ok := v.byID[c.Param("id")]
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "unknown battery"})
			return
		}

		status := battery.status()
		if status == nil {
			c.JSON(http.StatusNoContent, gin.H{})
			return
		}
		c.JSON(http.StatusOK, status)
	}
}

func (v *Controller) InfoGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		battery, ok := v.byID[c.Param("id")]
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "unknown battery"})
			return
		}

		info := battery.info()
		if info == nil {
			c.JSON(http.StatusNoContent, gin.H{})
			return
		}
		c.JSON(http.StatusOK, info)
	}
}

func (v *Controller) RegisterEndpoints(r *gin.Engine) {
	if len(v.batteries) == 0 {
		return
	}

	prefix := fmt.Sprintf("/api/%s", namespace)

	r.GET(prefix, v.IndexGet())
	r.GET(fmt.Sprintf("%s/:id/metrics", prefix), v.MetricsGet())
	r.GET(fmt.Sprintf("%s/:id/info", prefix), v.InfoGet())
}

func (v *Controller) Enabled() bool {
	return len(v.batteries) > 0
}

// Close stops collection, disconnects every battery, and then releases the
// shared BLE adapter once.
func (v *Controller) Close() error {
	if v.scheduler != nil {
		v.scheduler.Stop()
		log.Debug("voltgo scheduler stopped")
	}

	var firstErr error
	for _, battery := range v.batteries {
		if err := battery.close(); err != nil {
			log.Errorf("failed to close voltgo battery %s: %v", battery.id, err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	if v.connector != nil {
		if err := v.connector.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}
