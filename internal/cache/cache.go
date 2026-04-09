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

// LastUpdated reutrns time of most recent successful update call.
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
