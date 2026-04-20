// Package calculator computes provider ImpactResults.
package calculator

import (
	"fmt"
	"time"

	"github.com/eco-digit/leaf/internal/collector"
	"github.com/eco-digit/leaf/internal/infrastructure"
	"github.com/eco-digit/leaf/internal/intensity"
	"github.com/eco-digit/leaf/internal/model"
)

// TODO orchestrator.reporting_interval: "1h"
const (
	windowHours   = 1
	reportingUnit = "kwh"
)

// deviceEnergyKWh returns the energy in kWh for a single device over the
// reporting.interval.
func deviceEnergyKWh(d *collector.DeviceRaw) (kwh float64, fallback bool) {
	if bmc, ok := d.Metrics["bmc"]; ok && bmc > 0 {
		return bmc * windowHours / 1000.0, false
	}

	// kepler fallback
	idle := d.Metrics["kepler_node_idle"]
	active := d.Metrics["kepler_node_active"]
	if idle > 0 || active > 0 {
		return (idle + active) / 3600.0 / 1000.0, true
	}

	return kwh, false
}

// deviceEnergyResults creates an energy ImpatResult per device.
func deviceEnergyResults(
	raw *collector.RawMetrics,
	infra *infrastructure.Infrastructure,
	ts time.Time,
) (model.ResultSet, []string) {
	var rs model.ResultSet
	var warnings []string

	datacenter := infra.Environment.ID
	provider := infra.Environment.ID

	for _, dev := range infra.Devices {
		d, ok := raw.Devices[dev.ID]
		if !ok {
			continue
		}
		kwh, fallback := deviceEnergyKWh(d)
		if !ok {
			warnings = append(warnings, dev.ID+": no energy metric available (BMC and Kepler both missing)")
			continue
		}

		if fallback {
			warnings = append(warnings, dev.ID+": BMC missing, used Kepler idle+active as energy fallback source")
		}
		rs = append(rs, model.ImpactResult{
			Subject:     model.SubjectDevice,
			Provider:    provider,
			Datacenter:  datacenter,
			Component:   dev.Component,
			Device:      dev.ID,
			ImpactPhase: model.PhaseOperational,
			Category:    model.CategoryEnergy,
			Value:       kwh,
			Unit:        reportingUnit,
			Timestamp:   ts,
			PeriodHours: 1,
		})
	}

	return rs, warnings
}

// aggregateEnergyByComponent sums device energy results into provider-level
// component and total energy records.
func aggregateEnergyByComponent(
	deviceResults model.ResultSet,
	datacenter, provider string,
	ts time.Time,
) model.ResultSet {
	componentSum := make(map[string]float64)
	for _, r := range deviceResults {
		componentSum[r.Component] += r.Value
	}

	var rs model.ResultSet
	total := 0.0
	for component, kwh := range componentSum {
		rs = append(rs, model.ImpactResult{
			Subject:     model.SubjectProvider,
			Provider:    provider,
			Datacenter:  datacenter,
			Component:   component,
			ImpactPhase: model.PhaseOperational,
			Category:    model.CategoryEnergy,
			Value:       kwh,
			Unit:        reportingUnit,
			Timestamp:   ts,
			PeriodHours: windowHours,
		})
		total += kwh
	}

	rs = append(rs, model.ImpactResult{
		Subject:     model.SubjectProvider,
		Provider:    provider,
		Datacenter:  datacenter,
		Component:   "total",
		ImpactPhase: model.PhaseOperational,
		Category:    model.CategoryEnergy,
		Value:       total,
		Unit:        reportingUnit,
		Timestamp:   ts,
		PeriodHours: windowHours,
	})

	return rs
}

// operationalImpactByComponent computes provider-level operational impact
// records for GWP, ADP, and CED from component-level energy records.
// The GWP factor is in g CO₂eq/kWh and is converted to kg CO₂eq.
// A category is skipped when its factor is zero (e.g. operational water has no
// factor yet in V1).
func operationalImpactByComponent(
	energyRS model.ResultSet,
	pue float64,
	factors intensity.IntensityFactors,
	ts time.Time,
) model.ResultSet {
	type spec struct {
		cat    model.Category
		factor float64
		unit   string
		scale  float64 // applied after E × PUE × factor
	}
	specs := []spec{
		{model.CategoryGWP, factors.GWP.Value, "kg_co2eq", 1.0 / 1000.0}, // g → kg
		{model.CategoryADP, factors.ADP.Value, "kg_sb_eq", 1.0},
		{model.CategoryCED, factors.CED.Value, "mj", 1.0},
	}

	var rs model.ResultSet
	for _, r := range energyRS {
		for _, s := range specs {
			if s.factor == 0 {
				continue
			}
			rs = append(rs, model.ImpactResult{
				Subject:     model.SubjectProvider,
				Provider:    r.Provider,
				Datacenter:  r.Datacenter,
				Component:   r.Component,
				ImpactPhase: model.PhaseOperational,
				Category:    s.cat,
				Value:       r.Value * pue * s.factor * s.scale,
				Unit:        s.unit,
				Timestamp:   ts,
				PeriodHours: windowHours,
			})
		}
	}
	return rs
}

// totalImpactByComponent merges provider-level operational and embodied records
// into PhaseTotal records. Categories that appear only in embodied (e.g. Water
// in V1) are included with operational == 0.
func totalImpactByComponent(
	operationalRS model.ResultSet,
	embodiedRS model.ResultSet,
	ts time.Time,
) model.ResultSet {
	type key struct {
		component string
		cat       model.Category
	}
	type entry struct {
		operational float64
		embodied    float64
		unit        string
		provider    string
		datacenter  string
	}

	totals := make(map[key]*entry)

	for _, r := range operationalRS.FilterBySubject(model.SubjectProvider) {
		k := key{r.Component, r.Category}
		if totals[k] == nil {
			totals[k] = &entry{provider: r.Provider, datacenter: r.Datacenter, unit: r.Unit}
		}
		totals[k].operational += r.Value
	}

	for _, r := range embodiedRS.FilterBySubject(model.SubjectProvider) {
		k := key{r.Component, r.Category}
		if totals[k] == nil {
			totals[k] = &entry{provider: r.Provider, datacenter: r.Datacenter, unit: r.Unit}
		}
		totals[k].embodied += r.Value
	}

	var rs model.ResultSet
	for k, e := range totals {
		rs = append(rs, model.ImpactResult{
			Subject:     model.SubjectProvider,
			Provider:    e.provider,
			Datacenter:  e.datacenter,
			Component:   k.component,
			ImpactPhase: model.PhaseTotal,
			Category:    k.cat,
			Value:       e.operational + e.embodied,
			Unit:        e.unit,
			Timestamp:   ts,
			PeriodHours: windowHours,
		})
	}
	return rs
}

// validateEnergy checks that device energy sums to component
// energy and component energy sums to total.
// TODO maybe move to a validation section later for now in WIP
//
//	 status helpful for testing and validation while dev
//
//		Returns warnings for each violation.
func validateEnergy(rs model.ResultSet) []string {
	energy := rs.FilterByCategory(model.CategoryEnergy)

	deviceSum := make(map[string]float64)
	for _, r := range energy.FilterBySubject(model.SubjectDevice) {
		deviceSum[r.Component] += r.Value
	}

	var warnings []string
	componentSum := 0.0
	for _, r := range energy.FilterBySubject(model.SubjectProvider) {
		if r.Component == "total" {
			continue
		}
		if expected := deviceSum[r.Component]; r.Value != expected {
			warnings = append(warnings, fmt.Sprintf(
				"energy component %s: provider %.6f != sum of devices %.6f",
				r.Component, r.Value, expected,
			))
		}
		componentSum += r.Value
	}

	for _, r := range energy.FilterBySubject(model.SubjectProvider).FilterByComponent("total") {
		if r.Value != componentSum {
			warnings = append(warnings, fmt.Sprintf(
				"energy total: provider %.6f != sum of components %.6f",
				r.Value, componentSum,
			))
		}
	}

	return warnings
}

// ProviderResults runs the full provider-level calculation pipeline:
// device energy > component energy > operational impacts > total impacts.
//
// Embodied records loaded in cash on startup are passed in to produce totals
// but are not re-emitted — caller merges them from the cache.
func ProviderResults(
	raw *collector.RawMetrics,
	infra *infrastructure.Infrastructure,
	factors intensity.IntensityFactors,
	embodiedRS model.ResultSet,
	ts time.Time,
) (model.ResultSet, []string) {
	datacenter := infra.Environment.ID
	provider := infra.Environment.ID
	pue := infra.Environment.PUE

	deviceEnergy, warnings := deviceEnergyResults(raw, infra, ts)
	componentEnergy := aggregateEnergyByComponent(deviceEnergy, datacenter, provider, ts)
	operational := operationalImpactByComponent(componentEnergy, pue, factors, ts)
	totals := totalImpactByComponent(operational, embodiedRS, ts)

	// TODO move validation
	validationWarnings := validateEnergy(append(deviceEnergy, componentEnergy...))
	warnings = append(warnings, validationWarnings...)

	var rs model.ResultSet
	rs = append(rs, deviceEnergy...)
	rs = append(rs, componentEnergy...)
	rs = append(rs, operational...)
	rs = append(rs, totals...)
	return rs, warnings
}
