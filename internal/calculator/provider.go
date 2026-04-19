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

// deviceEnergyKWh returns the energie in kWh for a single device over the
// reporting.interval.
func deviceEnergyKWh(d *collector.DeviceRaw) (kwh float64, keplerFallback bool) {
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

// deviceEnergyResults creates a energy ImpatResult per device.
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
		if kwh == 0 {
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
