package cache

import (
	"testing"
	"time"

	"github.com/eco-digit/leaf/internal/model"
)

func makeResults(n int) model.ResultSet {
	rs := make(model.ResultSet, n)
	for i := range rs {
		rs[i] = model.ImpactResult{
			Subject:     model.SubjectProvider,
			Category:    model.CategoryGWP,
			ImpactPhase: model.PhaseEmbodied,
			Value:       float64(i),
			Unit:        "kg_co2eq",
			Timestamp:   time.Now(),
			PeriodHours: 1,
		}
	}
	return rs
}

func TestNewCacheIsEmpty(t *testing.T) {
	c := New()
	if !c.IsEmpty() {
		t.Error("new cache should be empty")
	}
	if snap := c.Snapshot(); snap != nil {
		t.Errorf("Snapshot cache empty: got %v, wanted nil", snap)
	}
	if !c.LastUpdated().IsZero() {
		t.Error("LastUpdate on empty cache should be zero")
	}
}
