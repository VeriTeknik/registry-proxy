package cache

import (
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/veriteknik/registry-proxy/internal/models"
)

// Cache manages caching of enriched server data
type Cache struct {
	store      *cache.Cache
	mu         sync.RWMutex
	lastUpdate time.Time
}

// NewCache creates a new cache instance
func NewCache(defaultExpiration, cleanupInterval time.Duration) *Cache {
	return &Cache{
		store: cache.New(defaultExpiration, cleanupInterval),
	}
}

const (
	// AllServersKey is the cache key for all enriched servers
	AllServersKey = "all_servers_enriched"
)

// SetServers caches the enriched server list
func (c *Cache) SetServers(servers []models.EnrichedServer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.store.Set(AllServersKey, servers, cache.DefaultExpiration)
	c.lastUpdate = time.Now()
}

// GetServers retrieves cached servers if available
func (c *Cache) GetServers() ([]models.EnrichedServer, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if data, found := c.store.Get(AllServersKey); found {
		if servers, ok := data.([]models.EnrichedServer); ok {
			return servers, true
		}
	}
	return nil, false
}

// GetLastUpdate returns when the cache was last updated
func (c *Cache) GetLastUpdate() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastUpdate
}

// Clear removes all cached data
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store.Flush()
}