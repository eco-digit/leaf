// Package cache provides a thread-safe store for Leaf's pre-calculated metrics.

package cache

import (
	"sync"
	"time"

	"github.com/OSBA-eco-digit/leaf/internal/model"
)

// Cache holds the most recently calculated ResultSet together with the time it was last updated.
type Cache struct {
	mu          sync.RWMutex
	results     model.ResultSet
	lastUpdated time.Time
}

// New returns an emptycache.
func New() *Cache {
	return &Cache{}
}

// Snapshot return a copy of the ResultsSet.
func (c *Cache) Snapshot() model.ResultSet {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.results) == 0 {
		return nil
	}
	snap := make(model.ResultSet, len(c.results))
	copy(snap, c.results)
	return snap
}

// Update replaces stored ResultsSet with rs and records the time as last Updated timestamp.
// TODO this will be called by the model calculator after each succ. calc cycle.
func (c *Cache) Update(rs model.ResultSet) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results = rs
	c.lastUpdated = time.Now()
}

// LastUpdated returns time of most recent successful update call.
func (c *Cache) LastUpdated() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastUpdated
}

// IsEmpty shows if cache is empty.
func (c *Cache) IsEmpty() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.results) == 0
}
