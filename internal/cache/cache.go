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
