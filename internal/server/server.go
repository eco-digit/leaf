// Package server exposes the pre-calculated metrics over /metrics.
package server

import (
	"github.com/OSBA-eco-digit/leaf/internal/cache"
)

// Server is the http server of leaf.
type Server struct {
	cache *cache.Cache
	addr  string
}

// New creates a server that reads from c and listens on addr.
func New(c *cache.Cache, addr string) *Server {
	return &Server{cache: c, addr: addr}
}
