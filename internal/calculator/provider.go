// Package calculator computes provider ImpactResults.
package calculator

import (
	"time"

	"github.com/eco-digit/leaf/internal/collector"
	"github.com/eco-digit/leaf/internal/infrastructure"
	"github.com/eco-digit/leaf/internal/model"
)

// TODO orchestrator.reporting_interval: "1h"
const windowHours = 1

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
		return (idle + active) / 3600.0 / 1000.0, trueß
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
			Unit:        "kwh",
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
			Unit:        "kwh",
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
		Unit:        "kwh",
		Timestamp:   ts,
		PeriodHours: windowHours,
	})

	return rs
}
