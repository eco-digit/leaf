// Package embodied computes per hour amortized embodied ImpactResults (static)
// from device profiles at startup.
package embodied

import (
	"fmt"
	"math"
	"time"

	"github.com/eco-digit/leaf/internal/infrastructure"
	"github.com/eco-digit/leaf/internal/model"
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

// Validate checks that embodied results are internally consistent across the
// aggregation hierarchy. We check: device -> component -> provider total For
// each impact category:
// 1. Sum of all devices valu of a coomponent must be
// equal the total provider component value.
// 2. Sum of all component level
// provider value equal provider total.
func Validate(rs model.ResultSet) error {
	const tol = 1e-6

	embodied := rs.FilterByPhase(model.PhaseEmbodied)

	// Expected component sums from device records.
	// deviceSum[component][category] = sum of device values
	deviceSum := make(map[string]map[model.Category]float64)
	for _, r := range embodied.FilterBySubject(model.SubjectDevice) {
		if deviceSum[r.Component] == nil {
			deviceSum[r.Component] = make(map[model.Category]float64)
		}
		deviceSum[r.Component][r.Category] += r.Value
	}

	// Check component provider records against device sums.
	componentSum := make(map[model.Category]float64)
	for _, r := range embodied.FilterBySubject(model.SubjectProvider) {
		if r.Component == "total" {
			continue
		}
		expected := deviceSum[r.Component][r.Category]
		if math.Abs(r.Value-expected) > tol {
			return fmt.Errorf(
				"embodied component %s/%s: provider value %.6f != sum of devices %.6f (diff %.6f)",
				r.Component, r.Category, r.Value, expected, r.Value-expected,
			)
		}
		componentSum[r.Category] += r.Value
	}

	// Check total provider records against component sums.
	for _, r := range embodied.FilterBySubject(model.SubjectProvider).FilterByComponent("total") {
		expected := componentSum[r.Category]
		if math.Abs(r.Value-expected) > tol {
			return fmt.Errorf(
				"embodied total/%s: provider value %.6f != sum of components %.6f (diff %.6f)",
				r.Category, r.Value, expected, r.Value-expected,
			)
		}
	}

	return nil
}
