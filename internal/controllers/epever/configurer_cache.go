package epever

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
)

type cachedConfig struct {
	config    *ControllerConfig
	timestamp time.Time
}

// getCachedConfig returns the cached config if valid, otherwise fetches from device
func (sc *Configurer) getCachedConfig(ctx context.Context) (*ControllerConfig, error) {
	sc.cacheMutex.RLock()
	if sc.cache != nil && time.Since(sc.cache.timestamp) < sc.cacheTTL {
		// Return a copy to prevent external modification of cached data
		configCopy := *sc.cache.config
		sc.cacheMutex.RUnlock()
		log.Trace("Using cached config")
		return &configCopy, nil
	}
	sc.cacheMutex.RUnlock()

	// Cache miss or expired - fetch from device
	config, err := sc.getConfig(ctx)
	if err != nil {
		return nil, err
	}

	// Update cache with a copy
	configCopy := config
	sc.cacheMutex.Lock()
	sc.cache = &cachedConfig{
		config:    &configCopy,
		timestamp: time.Now(),
	}
	sc.cacheMutex.Unlock()

	log.Trace("Fetched and cached config from device")
	return &config, nil
}

// invalidateCache clears the cache, forcing the next read to fetch from device
func (sc *Configurer) invalidateCache() {
	sc.cacheMutex.Lock()
	sc.cache = nil
	sc.cacheMutex.Unlock()
	log.Trace("Config cache invalidated")
}
