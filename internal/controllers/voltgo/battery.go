package voltgo

import (
	"context"
	"fmt"
	"sync"

	"github.com/lumberbarons/solar-controller/internal/publish"
	log "github.com/sirupsen/logrus"
)

// batteryRunner owns everything specific to one battery: its BLE collector,
// its cached status and info for the HTTP handlers, and the id that
// distinguishes its routes, Prometheus series, and publisher topics from
// every other battery's.
//
// The Controller drives one runner per configured battery, so a battery that
// cannot be reached fails inside its own collect call and leaves the rest of
// the cycle to carry on.
type batteryRunner struct {
	id        string
	address   string
	collector *Collector
	publisher publish.MessagePublisher
	metrics   MetricsCollector
	deviceID  string

	mu         sync.RWMutex
	lastStatus *BatteryStatus
	lastInfo   *BatteryInfo
}

func newBatteryRunner(
	id string,
	address string,
	collector *Collector,
	publisher publish.MessagePublisher,
	metrics MetricsCollector,
	deviceID string,
) *batteryRunner {
	return &batteryRunner{
		id:        id,
		address:   address,
		collector: collector,
		publisher: publisher,
		metrics:   metrics,
		deviceID:  deviceID,
	}
}

// collectAndPublish reads this battery and publishes its metrics. It never
// returns an error: a battery that cannot be read reports a failure metric and
// leaves the others in the cycle unaffected.
func (b *batteryRunner) collectAndPublish(ctx context.Context) {
	log.Debugf("collecting and publishing metrics for voltgo battery %s", b.id)

	status, err := b.collector.GetStatus(ctx)
	if err != nil {
		log.Errorf("failed to collect metrics from voltgo battery %s: %s", b.id, err)
		b.metrics.IncrementFailures(b.id)
		b.publishMetric(CreateCollectionFailureMetric())
		return
	}

	b.mu.Lock()
	b.lastStatus = status
	b.mu.Unlock()

	b.metrics.SetMetrics(b.id, status)

	// Fetch static battery info once, now that a connection is up
	b.fetchInfoOnce(ctx)

	for _, metric := range ConvertStatusToMetrics(status) {
		b.publishMetric(metric)
	}

	log.Debugf("collection done for voltgo battery %s", b.id)
}

// publishMetric publishes one metric under this battery's topic.
//
// The battery id is its own segment rather than part of the metric name so a
// subscriber can still wildcard a metric across every battery
// (`solar/+/voltgo/+/battery-soc`) as well as every metric of one battery.
func (b *batteryRunner) publishMetric(metric Metric) {
	payload, err := metric.ToJSON()
	if err != nil {
		log.Errorf("failed to marshal metric %s for voltgo battery %s: %s", metric.Name, b.id, err)
		return
	}

	topicSuffix := fmt.Sprintf("%s/%s/%s/%s", b.deviceID, namespace, b.id, metric.Name)
	b.publisher.Publish(topicSuffix, payload)
	log.Debugf("published metric %s to %s", metric.Name, topicSuffix)
}

// fetchInfoOnce caches static battery info on the first successful collection
// cycle. Failures are logged but never fail the cycle - the next cycle retries.
func (b *batteryRunner) fetchInfoOnce(ctx context.Context) {
	b.mu.RLock()
	cached := b.lastInfo != nil
	b.mu.RUnlock()
	if cached {
		return
	}

	info, err := b.collector.GetInfo(ctx)
	if err != nil {
		log.Warnf("failed to read voltgo battery %s info: %s", b.id, err)
		return
	}

	b.mu.Lock()
	b.lastInfo = info
	b.mu.Unlock()
	log.Debugf("cached voltgo battery %s info: %s %.1fV %.0fAh",
		b.id, info.Chemistry, info.NominalVoltage, info.CapacityAh)
}

func (b *batteryRunner) status() *BatteryStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastStatus
}

func (b *batteryRunner) info() *BatteryInfo {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastInfo
}

func (b *batteryRunner) close() error {
	return b.collector.Close()
}
