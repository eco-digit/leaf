package cache

import (
	"time"

	"github.com/OSBA-eco-digit/leaf/internal/model"
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
