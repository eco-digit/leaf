// Package calculator computes provider ImpactResults.
package calculator

import (
	"time"

	"github.com/OSBA-eco-digit/leaf/internal/collector"
	"github.com/OSBA-eco-digit/leaf/internal/infrastructure"
	"github.com/OSBA-eco-digit/leaf/internal/model"
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

	return rs, warnings
}
