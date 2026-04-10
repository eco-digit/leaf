// Package embodied computes per hour amortized embodied ImpactResults (static)
// from device profiles at startup.
package embodied

import (
	"fmt"
	"time"

	"github.com/OSBA-eco-digit/leaf/internal/infrastructure"
	"github.com/OSBA-eco-digit/leaf/internal/model"
)

const (
	defaultLifespanYears = 4
	hoursPerYear         = 8760
)

// categorySpec binds a model.Category to its unit and the parser on
// EmbodiedImpact.
type categorySpec struct {
	cat   model.Category
	unit  string
	parse func(infrastructure.EmbodiedImpact) (float64, error)
}

var categories = []categorySpec{
	{model.CategoryGWP, "kg_co2eq", func(e infrastructure.EmbodiedImpact) (float64, error) { return e.ParseGWP() }},
	{model.CategoryADP, "kg_sb_eq", func(e infrastructure.EmbodiedImpact) (float64, error) { return e.ParseADP() }},
	{model.CategoryCED, "mj", func(e infrastructure.EmbodiedImpact) (float64, error) { return e.ParseCED() }},
	{model.CategoryWater, "m3", func(e infrastructure.EmbodiedImpact) (float64, error) { return e.ParseWater() }},
}

// Calculate returns embodied ImpactResults records derived from the
// infrastructure.
func Calculate(infra *infrastructure.Infrastructure, ts time.Time) (model.ResultSet, error) {
	var rs model.ResultSet
	datacenter := infra.Environment.ID
	provider := infra.Environment.ID

	componentTotals := make(map[string]map[model.Category]float64)

	for _, dev := range infra.Devices {
		if dev.Profile == nil {
			continue
		}
		lifespan := dev.Profile.DefaultLifespanYears
		if lifespan <= 0 {
			lifespan = defaultLifespanYears
		}
		divisor := float64(lifespan) * hoursPerYear

		for _, cs := range categories {
			total, err := cs.parse(dev.Profile.EmbodiedImpact)
			if err != nil {
				return nil, fmt.Errorf("device %s %s: %w", dev.ID, cs.cat, err)
			}
			perHour := total / divisor

			rs = append(rs, model.ImpactResult{
				Subject:     model.SubjectDevice,
				Provider:    provider,
				Datacenter:  datacenter,
				Component:   dev.Component,
				Device:      dev.ID,
				ImpactPhase: model.PhaseEmbodied,
				Category:    cs.cat,
				Value:       perHour,
				Unit:        cs.unit,
				Timestamp:   ts,
				PeriodHours: 1,
			})

			if componentTotals[dev.Component] == nil {
				componentTotals[dev.Component] = make(map[model.Category]float64)
			}
			componentTotals[dev.Component][cs.cat] += perHour
		}
	}

	// Per component and category.
	grandTotal := make(map[model.Category]float64)
	for component, catMap := range componentTotals {
		for _, cs := range categories {
			val := catMap[cs.cat]
			rs = append(rs, model.ImpactResult{
				Subject:     model.SubjectProvider,
				Provider:    provider,
				Datacenter:  datacenter,
				Component:   component,
				ImpactPhase: model.PhaseEmbodied,
				Category:    cs.cat,
				Value:       val,
				Unit:        cs.unit,
				Timestamp:   ts,
				PeriodHours: 1,
			})
			grandTotal[cs.cat] += val
		}
	}

	// Per category provider total
	for _, cs := range categories {
		rs = append(rs, model.ImpactResult{
			Subject:     model.SubjectProvider,
			Provider:    provider,
			Datacenter:  datacenter,
			Component:   "total",
			ImpactPhase: model.PhaseEmbodied,
			Category:    cs.cat,
			Value:       grandTotal[cs.cat],
			Unit:        cs.unit,
			Timestamp:   ts,
			PeriodHours: 1,
		})
	}

	return rs, nil
}
