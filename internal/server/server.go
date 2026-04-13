// Package server exposes the pre-calculated metrics over /metrics.
package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/OSBA-eco-digit/leaf/internal/cache"
)

// Server is the http server of leaf.
type Server struct {
	cache *cache.Cache
	addr  string
}

type healthResponse struct {
	Status      string    `json:"status"`
	LastUpdated time.Time `json:"last_updated,omitempty"`
	CacheEmpty  bool      `json:"cache_empty"`
	AgeSeconds  float64   `json:"cache_age_seconds,omitempty"`
}

// New creates a server that reads from c and listens on addr.
func New(c *cache.Cache, addr string) *Server {
	return &Server{cache: c, addr: addr}
}

// Handler returns http.Hanlder registered with /metrics.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", s.metricsHandler)
	return mux
}

// Start registers handlers.
func (s *Server) Start() error {
	return http.ListenAndServe(s.addr, s.Handler())
}

func (s *Server) metricsHandler(w http.ResponseWriter, _ *http.Request) {
	rs := s.cache.Snapshot()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	writeMetrics(w, rs)
}

func (s *Server) healthHandler(w http.ResponseWriter, _ *http.Request) {
	last := s.cache.LastUpdated()
	empty := s.cache.IsEmpty()

	resp := healthResponse{
		Status:     "ok",
		CacheEmpty: empty,
	}
	if !last.IsZero() {
		resp.LastUpdated = last
		resp.AgeSeconds = time.Since(last).Seconds()
	}
	if empty {
		resp.Status = "no_data"
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		return
	}
}
